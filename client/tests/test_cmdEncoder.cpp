#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include <cstring>

#if defined(_WIN32)
    #include <winsock2.h>
#else
    #include <arpa/inet.h>
#endif

#include "protocol/cmdEncoder.h"
#include "result.h"
#include "internalError.h"




class CommandEncoderTestWrapper : public Protocol::CommandEncoder {
public:
    using CommandEncoder::get_optimal_buffer_size;
    using CommandEncoder::to_field_id;
    using CommandEncoder::is_within_data_size;
    using CommandEncoder::append_uint8;
    using CommandEncoder::append_uint16;
    using CommandEncoder::append_uint32;
    using CommandEncoder::append_double;
    using CommandEncoder::append_string;
    using CommandEncoder::encode_service;
    using CommandEncoder::decode_service;
    using CommandEncoder::encode_account_number;
    using CommandEncoder::decode_account_number;
    using CommandEncoder::encode_account_owner_name;
    using CommandEncoder::decode_account_owner_name;
    using CommandEncoder::encode_account_password;
    using CommandEncoder::decode_account_password;
    using CommandEncoder::encode_tx_account_number;
    using CommandEncoder::decode_tx_account_number;
    using CommandEncoder::encode_tx_account_owner_name;
    using CommandEncoder::decode_tx_account_owner_name;
    using CommandEncoder::encode_monetary_value;
    using CommandEncoder::decode_monetary_value;
    using CommandEncoder::encode_currency;
    using CommandEncoder::decode_currency;
    using CommandEncoder::encode_monitor_updates;
    using CommandEncoder::decode_monitor_updates;
    using CommandEncoder::encode_monitor_timeout_seconds;
    using CommandEncoder::decode_monitor_timeout_seconds;
};

class CommandEncoderTest : public ::testing::Test {
protected:
    CommandEncoderTestWrapper encoder;
};


TEST_F(CommandEncoderTest, append_uint8) {
    std::vector<uint8_t> buffer;
    encoder.append_uint8(buffer, 0xAB);
    ASSERT_EQ(buffer.size(), 1);
    EXPECT_EQ(buffer[0], 0xAB);
}

TEST_F(CommandEncoderTest, append_uint16) {
    std::vector<uint8_t> buffer;
    encoder.append_uint16(buffer, 0x1234);
    ASSERT_EQ(buffer.size(), 2);
    uint16_t val;
    std::memcpy(&val, buffer.data(), 2);
    EXPECT_EQ(ntohs(val), 0x1234);
}

TEST_F(CommandEncoderTest, append_uint32) {
    std::vector<uint8_t> buffer;
    encoder.append_uint32(buffer, 0x12345678);
    ASSERT_EQ(buffer.size(), 4);
    uint32_t val;
    std::memcpy(&val, buffer.data(), 4);
    EXPECT_EQ(ntohl(val), 0x12345678);
}

TEST_F(CommandEncoderTest, append_double) {
    std::vector<uint8_t> buffer;
    double original = 123.456;
    encoder.append_double(buffer, original);
    ASSERT_EQ(buffer.size(), 8);
    
    uint64_t val;
    std::memcpy(&val, buffer.data(), 8);
    val = ((val & 0xFF00000000000000ULL) >> 56) |
          ((val & 0x00FF000000000000ULL) >> 40) |
          ((val & 0x0000FF0000000000ULL) >> 24) |
          ((val & 0x000000FF00000000ULL) >> 8)  |
          ((val & 0x00000000FF000000ULL) << 8)  |
          ((val & 0x0000000000FF0000ULL) << 24) |
          ((val & 0x000000000000FF00ULL) << 40) |
          ((val & 0x00000000000000FFULL) << 56);
    double decoded;
    std::memcpy(&decoded, &val, 8);
    EXPECT_DOUBLE_EQ(decoded, original);
}

TEST_F(CommandEncoderTest, append_string) {
    std::vector<uint8_t> buffer;
    std::string testStr = "Hello";
    encoder.append_string(buffer, testStr);
    ASSERT_EQ(buffer.size(), 5);
    EXPECT_EQ(std::string(buffer.begin(), buffer.end()), "Hello");
}

TEST_F(CommandEncoderTest, to_field_id_valid) {
    auto res = encoder.to_field_id(1);
    ASSERT_TRUE(res.has_value());
    EXPECT_EQ(*res, Protocol::FieldID::Service);
    
    res = encoder.to_field_id(8);
    ASSERT_TRUE(res.has_value());
    EXPECT_EQ(*res, Protocol::FieldID::Currency);
}

TEST_F(CommandEncoderTest, to_field_id_valid_monitor_fields) {
    auto res = encoder.to_field_id(9);
    ASSERT_TRUE(res.has_value());
    EXPECT_EQ(*res, Protocol::FieldID::MonitorUpdates);

    res = encoder.to_field_id(10);
    ASSERT_TRUE(res.has_value());
    EXPECT_EQ(*res, Protocol::FieldID::MonitorTimeoutSeconds);
}

TEST_F(CommandEncoderTest, to_field_id_invalid) {
    auto res = encoder.to_field_id(0);
    EXPECT_FALSE(res.has_value());

    res = encoder.to_field_id(11);
    EXPECT_FALSE(res.has_value());
}

TEST_F(CommandEncoderTest, is_within_data_size) {
    std::vector<uint8_t> data(10, 0);
    EXPECT_TRUE(encoder.is_within_data_size(0, 10, data));
    EXPECT_TRUE(encoder.is_within_data_size(5, 5, data));
    EXPECT_FALSE(encoder.is_within_data_size(5, 6, data));
    EXPECT_FALSE(encoder.is_within_data_size(11, 0, data));
}

TEST_F(CommandEncoderTest, get_optimal_buffer_size) {
    Protocol::Command cmd;
    cmd.service = Protocol::Service::DEPOSIT;
    cmd.account_number = 123;
    EXPECT_EQ(encoder.get_optimal_buffer_size(cmd), 15);
    
    cmd.account_owner_name = "Alice";
    EXPECT_EQ(encoder.get_optimal_buffer_size(cmd), 25);
}


TEST_F(CommandEncoderTest, encode_decode_service) {
    Protocol::Command cmd;
    cmd.service = Protocol::Service::DEPOSIT;
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_service(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    
    ASSERT_EQ(buffer.size(), 6);
    EXPECT_EQ(buffer[0], static_cast<uint8_t>(Protocol::FieldID::Service));
    
    Protocol::Command decoded_cmd;
    size_t offset = 5; // Start of value
    uint32_t length = 1;
    auto res_dec = encoder.decode_service(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    ASSERT_TRUE(decoded_cmd.service.has_value());
    EXPECT_EQ(decoded_cmd.service.value(), Protocol::Service::DEPOSIT);
}

TEST_F(CommandEncoderTest, encode_decode_account_number) {
    Protocol::Command cmd;
    cmd.account_number = 123456;
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_account_number(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    
    ASSERT_EQ(buffer.size(), 9);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 4;
    auto res_dec = encoder.decode_account_number(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    ASSERT_TRUE(decoded_cmd.account_number.has_value());
    EXPECT_EQ(decoded_cmd.account_number.value(), 123456);
}

TEST_F(CommandEncoderTest, encode_decode_account_owner_name) {
    Protocol::Command cmd;
    cmd.account_owner_name = "Alice";
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_account_owner_name(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 1 + 4 + 5);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 5;
    auto res_dec = encoder.decode_account_owner_name(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_EQ(decoded_cmd.account_owner_name.value(), "Alice");
}

TEST_F(CommandEncoderTest, encode_decode_account_password) {
    Protocol::Command cmd;
    cmd.account_password = "password123";
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_account_password(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 1 + 4 + 11);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 11;
    auto res_dec = encoder.decode_account_password(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_EQ(decoded_cmd.account_password.value(), "password123");
}

TEST_F(CommandEncoderTest, encode_decode_tx_account_number) {
    Protocol::Command cmd;
    cmd.tx_account_number = 987654;
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_tx_account_number(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 9);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 4;
    auto res_dec = encoder.decode_tx_account_number(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_EQ(decoded_cmd.tx_account_number.value(), 987654);
}

TEST_F(CommandEncoderTest, encode_decode_tx_account_owner_name) {
    Protocol::Command cmd;
    cmd.tx_account_owner_name = "Bob";
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_tx_account_owner_name(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 3;
    auto res_dec = encoder.decode_tx_account_owner_name(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_EQ(decoded_cmd.tx_account_owner_name.value(), "Bob");
}

TEST_F(CommandEncoderTest, encode_decode_monetary_value) {
    Protocol::Command cmd;
    cmd.monetary_value = 1234.56;
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_monetary_value(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 1 + 4 + 8);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 8;
    auto res_dec = encoder.decode_monetary_value(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_DOUBLE_EQ(decoded_cmd.monetary_value.value(), 1234.56);
}

TEST_F(CommandEncoderTest, encode_decode_currency) {
    Protocol::Command cmd;
    cmd.currency = Protocol::CurrencyType::USD;
    std::vector<uint8_t> buffer;
    
    auto res_enc = encoder.encode_currency(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 1 + 4 + 1);
    
    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 1;
    auto res_dec = encoder.decode_currency(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    EXPECT_EQ(decoded_cmd.currency.value(), Protocol::CurrencyType::USD);
}

TEST_F(CommandEncoderTest, encode_decode_monitor_updates) {
    Protocol::Command cmd;
    cmd.monitor_updates = "account_updated";
    std::string value = cmd.monitor_updates.value();
    uint32_t str_len = static_cast<uint32_t>(value.size());
    std::vector<uint8_t> buffer;

    auto res_enc = encoder.encode_monitor_updates(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 1 + 4 + str_len);
    EXPECT_EQ(buffer[0], static_cast<uint8_t>(Protocol::FieldID::MonitorUpdates));

    Protocol::Command decoded_cmd;
    size_t offset = 5;
    auto res_dec = encoder.decode_monitor_updates(decoded_cmd, offset, str_len, buffer);
    ASSERT_TRUE(res_dec.ok());
    ASSERT_TRUE(decoded_cmd.monitor_updates.has_value());
    EXPECT_EQ(decoded_cmd.monitor_updates.value(), "account_updated");
}

TEST_F(CommandEncoderTest, encode_decode_monitor_updates_empty_string) {
    Protocol::Command cmd;
    cmd.monitor_updates = "";
    std::vector<uint8_t> buffer;

    auto res_enc = encoder.encode_monitor_updates(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 5);

    Protocol::Command decoded_cmd;
    size_t offset = 5;
    auto res_dec = encoder.decode_monitor_updates(decoded_cmd, offset, 0, buffer);
    ASSERT_TRUE(res_dec.ok());
    ASSERT_TRUE(decoded_cmd.monitor_updates.has_value());
    EXPECT_EQ(decoded_cmd.monitor_updates.value(), "");
}

TEST_F(CommandEncoderTest, decode_monitor_updates_string_too_long) {
    Protocol::Command decoded_cmd;
    std::vector<uint8_t> dummy_buffer(100, 0);
    size_t offset = 0;
    uint32_t bad_length = 1025;
    auto res = encoder.decode_monitor_updates(decoded_cmd, offset, bad_length, dummy_buffer);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DECODE_STRING_TOO_LONG);
}

TEST_F(CommandEncoderTest, encode_decode_monitor_timeout_seconds) {
    Protocol::Command cmd;
    cmd.monitor_timeout_seconds = 300;
    std::vector<uint8_t> buffer;

    auto res_enc = encoder.encode_monitor_timeout_seconds(buffer, cmd);
    ASSERT_TRUE(res_enc.ok());
    ASSERT_EQ(buffer.size(), 9);
    EXPECT_EQ(buffer[0], static_cast<uint8_t>(Protocol::FieldID::MonitorTimeoutSeconds));

    Protocol::Command decoded_cmd;
    size_t offset = 5;
    uint32_t length = 4;
    auto res_dec = encoder.decode_monitor_timeout_seconds(decoded_cmd, offset, length, buffer);
    ASSERT_TRUE(res_dec.ok());
    ASSERT_TRUE(decoded_cmd.monitor_timeout_seconds.has_value());
    EXPECT_EQ(decoded_cmd.monitor_timeout_seconds.value(), 300u);
}

TEST_F(CommandEncoderTest, decode_monitor_timeout_seconds_wrong_length) {
    Protocol::Command decoded_cmd;
    std::vector<uint8_t> dummy_buffer(10, 0);
    size_t offset = 0;
    auto res = encoder.decode_monitor_timeout_seconds(decoded_cmd, offset, 3, dummy_buffer);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DECODE_FIELD_OVERFLOW);
}

// Integration Tests (encode_message / decode_message)

TEST_F(CommandEncoderTest, full_cycle_success) {
    Protocol::Command cmd;
    cmd.service = Protocol::Service::TRANSFER_FUNDS;
    cmd.account_number = 1111;
    cmd.tx_account_number = 2222;
    cmd.monetary_value = 500.50;
    cmd.account_owner_name = "Alice";
    cmd.currency = Protocol::CurrencyType::EUR;
    
    auto res_enc = encoder.encode_message(cmd);
    ASSERT_TRUE(res_enc.ok());
    std::vector<uint8_t> encoded = res_enc.value();
    
    auto res_dec = encoder.decode_message(encoded);
    ASSERT_TRUE(res_dec.ok());
    Protocol::Command decoded = res_dec.value();
    
    EXPECT_EQ(decoded.service, cmd.service);
    EXPECT_EQ(decoded.account_number, cmd.account_number);
    EXPECT_EQ(decoded.tx_account_number, cmd.tx_account_number);
    EXPECT_DOUBLE_EQ(decoded.monetary_value.value(), cmd.monetary_value.value());
    EXPECT_EQ(decoded.account_owner_name, cmd.account_owner_name);
    EXPECT_EQ(decoded.currency, cmd.currency);
}

TEST_F(CommandEncoderTest, full_cycle_monitor_success) {
    Protocol::Command cmd;
    cmd.service = Protocol::Service::MONITOR;
    cmd.account_number = 5555;
    cmd.monitor_updates = "balance_changed";
    cmd.monitor_timeout_seconds = 60;

    auto res_enc = encoder.encode_message(cmd);
    ASSERT_TRUE(res_enc.ok());
    std::vector<uint8_t> encoded = res_enc.value();

    auto res_dec = encoder.decode_message(encoded);
    ASSERT_TRUE(res_dec.ok());
    Protocol::Command decoded = res_dec.value();

    EXPECT_EQ(decoded.service, cmd.service);
    EXPECT_EQ(decoded.account_number, cmd.account_number);
    ASSERT_TRUE(decoded.monitor_updates.has_value());
    EXPECT_EQ(decoded.monitor_updates.value(), "balance_changed");
    ASSERT_TRUE(decoded.monitor_timeout_seconds.has_value());
    EXPECT_EQ(decoded.monitor_timeout_seconds.value(), 60u);
}

TEST_F(CommandEncoderTest, encode_empty_fail) {
    Protocol::Command cmd; // No fields set
    auto res = encoder.encode_message(cmd);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::ENCODE_EMPTY_COMMAND);
}

TEST_F(CommandEncoderTest, decode_empty_fail) {
    std::vector<uint8_t> data;
    auto res = encoder.decode_message(data);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DECODE_EMPTY_DATA);
}

