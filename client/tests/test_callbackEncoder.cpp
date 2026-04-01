#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include <cstring>

#if defined(_WIN32)
    #include <winsock2.h>
#else
    #include <arpa/inet.h>
#endif

#include "protocol/callbackEncoder.h"
#include "result.h"
#include "internalError.h"

/*
    This test file tests the following functions:
    -----------------------------------------------

    validate_payload(size_t total_size);
    encode_message(const CallbackMessage& cb_message);
    decode_message(const std::vector<uint8_t>& data);

    Wire format (fixed layout, no TLV):
      [type     : 1 byte ]
      [service  : 1 byte ]
      [acc_num  : 4 bytes] (network order)
      [name_len : 4 bytes] (network order)
      [name     : N bytes]
      [currency : 1 byte ]
      [monetary : 8 bytes] (manual 64-bit byte-swap)

    MIN_CALLBACK_SIZE = 19 bytes (empty name, N=0)
*/

class CallbackEncoderTestWrapper : public Protocol::CallbackEncoder {
public:
    using CallbackEncoder::validate_payload;
};

class CallbackEncoderTest : public ::testing::Test {
protected:
    CallbackEncoderTestWrapper encoder;

    // Helper: build a minimal valid CallbackMessage
    Protocol::CallbackMessage make_msg(
        Protocol::Service      svc     = Protocol::Service::DEPOSIT,
        uint32_t               acc_num = 0,
        Protocol::CurrencyType cur     = Protocol::CurrencyType::SGD,
        double                 amount  = 0.0)
    {
        Protocol::CallbackMessage m{};
        m.type                   = Protocol::MessageType::Callback;
        m.service                = svc;
        m.account_number         = acc_num;
        m.account_owner_name_len = 0;
        m.account_owner_name     = "";
        m.currency               = cur;
        m.monetary_value         = amount;
        return m;
    }
};

TEST_F(CallbackEncoderTest, validate_payload_exact_minimum) {
    EXPECT_TRUE(encoder.validate_payload(MIN_CALLBACK_SIZE));
}

TEST_F(CallbackEncoderTest, validate_payload_too_short) {
    EXPECT_FALSE(encoder.validate_payload(MIN_CALLBACK_SIZE - 1));
    EXPECT_FALSE(encoder.validate_payload(0));
}

TEST_F(CallbackEncoderTest, validate_payload_larger_than_minimum) {
    EXPECT_TRUE(encoder.validate_payload(MIN_CALLBACK_SIZE + 1));
    EXPECT_TRUE(encoder.validate_payload(100));
}

TEST_F(CallbackEncoderTest, decode_message_empty_data) {
    std::vector<uint8_t> data;
    auto result = encoder.decode_message(data);
    ASSERT_FALSE(result.ok());
    EXPECT_EQ(result.error(), Error::InternalError::DESERIALIZE_PAYLOAD_OVERFLOW);
}

TEST_F(CallbackEncoderTest, decode_message_one_byte_short) {
    std::vector<uint8_t> data(MIN_CALLBACK_SIZE - 1, 0);
    auto result = encoder.decode_message(data);
    ASSERT_FALSE(result.ok());
    EXPECT_EQ(result.error(), Error::InternalError::DESERIALIZE_PAYLOAD_OVERFLOW);
}

TEST_F(CallbackEncoderTest, encode_message_empty_name_size) {
    auto result = encoder.encode_message(make_msg());
    ASSERT_TRUE(result.ok());
    // 1 + 1 + 4 + 4 + 0 + 1 + 8 = 19 = MIN_CALLBACK_SIZE
    EXPECT_EQ(result.value().size(), static_cast<size_t>(MIN_CALLBACK_SIZE));
}

TEST_F(CallbackEncoderTest, encode_message_with_name_size) {
    auto m = make_msg();
    m.account_owner_name_len = 5;
    m.account_owner_name     = "Alice";

    auto result = encoder.encode_message(m);
    ASSERT_TRUE(result.ok());
    EXPECT_EQ(result.value().size(), static_cast<size_t>(MIN_CALLBACK_SIZE + 5));
}

TEST_F(CallbackEncoderTest, encode_message_type_byte) {
    auto result = encoder.encode_message(make_msg());
    ASSERT_TRUE(result.ok());
    EXPECT_EQ(result.value()[0], static_cast<uint8_t>(Protocol::MessageType::Callback));
}

TEST_F(CallbackEncoderTest, encode_message_service_byte) {
    auto result = encoder.encode_message(make_msg(Protocol::Service::WITHDRAW));
    ASSERT_TRUE(result.ok());
    EXPECT_EQ(result.value()[1], static_cast<uint8_t>(Protocol::Service::WITHDRAW));
}

TEST_F(CallbackEncoderTest, encode_message_account_number_network_order) {
    auto m = make_msg(Protocol::Service::DEPOSIT, 0x12345678);
    auto result = encoder.encode_message(m);
    ASSERT_TRUE(result.ok());

    uint32_t raw;
    std::memcpy(&raw, result.value().data() + 2, 4);
    EXPECT_EQ(ntohl(raw), 0x12345678u);
}

TEST_F(CallbackEncoderTest, encode_message_name_len_network_order) {
    auto m = make_msg();
    m.account_owner_name_len = 5;
    m.account_owner_name     = "Alice";

    auto result = encoder.encode_message(m);
    ASSERT_TRUE(result.ok());

    uint32_t raw;
    std::memcpy(&raw, result.value().data() + 6, 4);
    EXPECT_EQ(ntohl(raw), 5u);
}

TEST_F(CallbackEncoderTest, encode_message_name_bytes) {
    auto m = make_msg();
    m.account_owner_name_len = 3;
    m.account_owner_name     = "Bob";

    auto result = encoder.encode_message(m);
    ASSERT_TRUE(result.ok());

    // Name starts at byte offset 10
    const auto& bytes = result.value();
    EXPECT_EQ(bytes[10], 'B');
    EXPECT_EQ(bytes[11], 'o');
    EXPECT_EQ(bytes[12], 'b');
}

TEST_F(CallbackEncoderTest, encode_message_currency_byte_position) {
    // With empty name, currency is at offset 10
    auto m = make_msg(Protocol::Service::DEPOSIT, 0, Protocol::CurrencyType::EUR);
    auto result = encoder.encode_message(m);
    ASSERT_TRUE(result.ok());
    EXPECT_EQ(result.value()[10], static_cast<uint8_t>(Protocol::CurrencyType::EUR));
}

TEST_F(CallbackEncoderTest, full_cycle_empty_name) {
    auto original = make_msg(Protocol::Service::DEPOSIT, 98765, Protocol::CurrencyType::USD, 1500.75);

    auto enc = encoder.encode_message(original);
    ASSERT_TRUE(enc.ok());

    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());

    EXPECT_EQ(dec.value().type,                   original.type);
    EXPECT_EQ(dec.value().service,                original.service);
    EXPECT_EQ(dec.value().account_number,         original.account_number);
    EXPECT_EQ(dec.value().account_owner_name_len, 0u);
    EXPECT_EQ(dec.value().account_owner_name,     "");
    EXPECT_EQ(dec.value().currency,               original.currency);
    EXPECT_DOUBLE_EQ(dec.value().monetary_value,  original.monetary_value);
}

TEST_F(CallbackEncoderTest, full_cycle_with_name) {
    auto original = make_msg(Protocol::Service::WITHDRAW, 11111, Protocol::CurrencyType::EUR, 250.00);
    original.account_owner_name_len = 3;
    original.account_owner_name     = "Bob";

    auto enc = encoder.encode_message(original);
    ASSERT_TRUE(enc.ok());

    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());

    EXPECT_EQ(dec.value().type,                   original.type);
    EXPECT_EQ(dec.value().service,                original.service);
    EXPECT_EQ(dec.value().account_number,         original.account_number);
    EXPECT_EQ(dec.value().account_owner_name_len, 3u);
    EXPECT_EQ(dec.value().account_owner_name,     "Bob");
    EXPECT_EQ(dec.value().currency,               original.currency);
    EXPECT_DOUBLE_EQ(dec.value().monetary_value,  original.monetary_value);
}

TEST_F(CallbackEncoderTest, full_cycle_long_name) {
    std::string long_name(50, 'X');
    auto original = make_msg(Protocol::Service::TRANSFER_FUNDS, 42000, Protocol::CurrencyType::SGD, 9999.99);
    original.account_owner_name_len = static_cast<uint32_t>(long_name.size());
    original.account_owner_name     = long_name;

    auto enc = encoder.encode_message(original);
    ASSERT_TRUE(enc.ok());
    EXPECT_EQ(enc.value().size(), static_cast<size_t>(MIN_CALLBACK_SIZE + 50));

    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());

    EXPECT_EQ(dec.value().account_number,         42000u);
    EXPECT_EQ(dec.value().account_owner_name,     long_name);
    EXPECT_EQ(dec.value().account_owner_name_len, 50u);
    EXPECT_DOUBLE_EQ(dec.value().monetary_value,  9999.99);
}

TEST_F(CallbackEncoderTest, full_cycle_zero_monetary_value) {
    auto original = make_msg(Protocol::Service::DEPOSIT, 1, Protocol::CurrencyType::SGD, 0.0);

    auto enc = encoder.encode_message(original);
    ASSERT_TRUE(enc.ok());

    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());
    EXPECT_DOUBLE_EQ(dec.value().monetary_value, 0.0);
}

TEST_F(CallbackEncoderTest, full_cycle_various_services) {
    for (auto svc : {Protocol::Service::DEPOSIT,
                     Protocol::Service::WITHDRAW,
                     Protocol::Service::TRANSFER_FUNDS}) {
        auto m = make_msg(svc, 55555, Protocol::CurrencyType::SGD, 100.0);
        auto enc = encoder.encode_message(m);
        ASSERT_TRUE(enc.ok());
        auto dec = encoder.decode_message(enc.value());
        ASSERT_TRUE(dec.ok());
        EXPECT_EQ(dec.value().service, svc);
    }
}

TEST_F(CallbackEncoderTest, full_cycle_various_currencies) {
    for (auto cur : {Protocol::CurrencyType::SGD,
                     Protocol::CurrencyType::USD,
                     Protocol::CurrencyType::EUR}) {
        auto m = make_msg(Protocol::Service::DEPOSIT, 0, cur, 50.0);
        auto enc = encoder.encode_message(m);
        ASSERT_TRUE(enc.ok());
        auto dec = encoder.decode_message(enc.value());
        ASSERT_TRUE(dec.ok());
        EXPECT_EQ(dec.value().currency, cur);
    }
}

TEST_F(CallbackEncoderTest, full_cycle_account_number_boundary) {
    auto m = make_msg(Protocol::Service::DEPOSIT, 0xFFFFFFFF, Protocol::CurrencyType::USD, 1.0);
    auto enc = encoder.encode_message(m);
    ASSERT_TRUE(enc.ok());
    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());
    EXPECT_EQ(dec.value().account_number, 0xFFFFFFFFu);
}

TEST_F(CallbackEncoderTest, full_cycle_monetary_precision) {
    // Verify fractional precision is preserved through the 64-bit byte-swap encoding
    auto original = make_msg(Protocol::Service::DEPOSIT, 0, Protocol::CurrencyType::USD, 123.456789);

    auto enc = encoder.encode_message(original);
    ASSERT_TRUE(enc.ok());

    auto dec = encoder.decode_message(enc.value());
    ASSERT_TRUE(dec.ok());
    EXPECT_DOUBLE_EQ(dec.value().monetary_value, 123.456789);
}
