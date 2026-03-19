#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "bankClient.h"
#include "bankIO.h"
#include "result.h"
#include "internalError.h"

/*

    This test file test for the following functions: 
    -----------------------------------------------

    isValidString(const std::string& str);
    isValidStringLength(const std::string& str);

    getValidatedString(const std::string& prompt);
    getValidatedPassword(const std::string& prompt);
    getValidatedCurrency(const std::string& prompt);
    getValidatedNumber(const std::string& prompt);
    
    fill_account_creation_details(Protocol::Command& req);
    fill_auth_details(Protocol::Command& req);
    fill_currency_details(Protocol::Command& req);
    fill_amount_details(Protocol::Command& req);
    fill_transfer_account_details(Protocol::Command& req);

    collect_user_input();  

*/

class MockBankIO : public BankIO{
public:
    MOCK_METHOD(std::string, read_line, (), (override));
    MOCK_METHOD(int, read_int, (), (override));
    MOCK_METHOD(void, print_prompt, (const std::string&), (override));
    MOCK_METHOD(void, print_error, (const std::string&), (override));
    MOCK_METHOD(void, print, (const std::string&, Colour), (override));
    MOCK_METHOD(void, print_box_top, (), (override));
    MOCK_METHOD(void, print_box_bottom, (), (override));
    MOCK_METHOD(void, print_service_menu, (), (override));
    MOCK_METHOD(void, wait_for_enter, (), (override));
};

class MockSocket : public NetworkUtils::BaseSocket {
public:
    MockSocket() : NetworkUtils::BaseSocket("127.0.0.1", 8080) {
        local_ip_port = {0, 0};
    }
    MOCK_METHOD((Result<std::monostate, Error::InternalError>), send_message, (const std::vector<uint8_t>&), (override));
    MOCK_METHOD((Result<std::vector<uint8_t>, Error::InternalError>), receive_message, (), (override));
    MOCK_METHOD((Result<std::monostate, Error::InternalError>), bind_socket, (), (override));
    MOCK_METHOD((std::pair<uint32_t, uint16_t>), get_local_info, (), (override));
};

class MockEncoder : public Protocol::BaseCommandEncoder {
public:
    MOCK_METHOD((Result<std::vector<uint8_t>, Error::InternalError>), encode_message, (const Protocol::Command&), (override));
    MOCK_METHOD((Result<Protocol::Command, Error::InternalError>), decode_message, (const std::vector<uint8_t>&), (override));
};

class MockSerializer : public Protocol::BaseMessageSerializer {
public:
    MOCK_METHOD((Result<std::vector<uint8_t>, Error::InternalError>), serialize, (const Protocol::Message&), (override));
    MOCK_METHOD((Result<Protocol::Message, Error::InternalError>), deserialize, (const std::vector<uint8_t>&), (override));
};

class BankClientTestWrapper : public BankClient {
public:
    BankClientTestWrapper(
        std::unique_ptr<BankIO> io,
        std::unique_ptr<NetworkUtils::BaseSocket> socket,
        std::unique_ptr<Protocol::BaseCommandEncoder> encoder,
        std::unique_ptr<Protocol::BaseMessageSerializer> serializer,
        Semantics::InvocationFlag flag
    ) : BankClient(std::move(io), std::move(socket), std::move(encoder), std::move(serializer), flag) {}
    
    // expose methods for testing
    using BankClient::isAlpha;
    using BankClient::isAlphaNumeric;
    using BankClient::isWithinMaxLength;

    using BankClient::getValidatedString;
    using BankClient::getValidatedPassword;
    using BankClient::getValidatedCurrency;
    using BankClient::getValidatedNumber;
    
    using BankClient::fill_account_creation_details;
    using BankClient::fill_auth_details;
    using BankClient::fill_currency_details;
    using BankClient::fill_amount_details;
    using BankClient::fill_transfer_account_details;

    using BankClient::collect_user_input;
    using BankClient::send_to_server;

    Protocol::BaseCommandEncoder* get_encoder() { return cmdEncoder.get(); }
    Protocol::BaseMessageSerializer* get_serializer() { return msgSerializer.get(); }
    NetworkUtils::BaseSocket* get_socket() { return socket.get(); }
};

class BankClientTest : public ::testing::Test {
protected:
    // Pointers to our mock and client. We use pointers so we can control initialization.
    MockBankIO* mockIO;                             // Raw pointer to set expectations
    std::unique_ptr<BankClientTestWrapper> client;  // The object we are testing
    
    void SetUp() override {
    
        auto uniqueMockIO = std::make_unique<MockBankIO>();
        auto uniqueMockSocket = std::make_unique<MockSocket>();
        auto uniqueMockEncoder = std::make_unique<MockEncoder>();
        auto uniqueMockSerializer = std::make_unique<MockSerializer>();
        
        mockIO = uniqueMockIO.get();
        
        // Initialize the client wrapper with all mocks
        client = std::make_unique<BankClientTestWrapper>(
            std::move(uniqueMockIO),
            std::move(uniqueMockSocket),
            std::move(uniqueMockEncoder),
            std::move(uniqueMockSerializer),
            Semantics::InvocationFlag::AT_LEAST_ONCE
        );
    }
   
    void TearDown() override {}
};


TEST_F(BankClientTest, IsAlpha_AcceptsValidStrings) {
    EXPECT_TRUE(client->isAlpha("John"));
    EXPECT_TRUE(client->isAlpha("Doe"));
    EXPECT_TRUE(client->isAlpha("ValidName"));
}

TEST_F(BankClientTest, IsAlpha_RejectsInvalidStrings) {
    EXPECT_FALSE(client->isAlpha(""));  
    EXPECT_FALSE(client->isAlpha("John123"));  
    EXPECT_FALSE(client->isAlpha("John Doe")); 
    EXPECT_FALSE(client->isAlpha("Jane~Doe")); 
}

TEST_F(BankClientTest, IsWithinMaxLength_ChecksLengthCorrectly) {
    // MAX_PW_LEN is 8
    EXPECT_TRUE(client->isWithinMaxLength("12345678")); 
    EXPECT_TRUE(client->isWithinMaxLength("123"));      
    
    EXPECT_FALSE(client->isWithinMaxLength("123456789")); 
    EXPECT_FALSE(client->isWithinMaxLength(""));      
}

TEST_F(BankClientTest, IsAlphaNumeric_AcceptsAlphanumeric) {
    EXPECT_TRUE(client->isAlphaNumeric("pass123"));
    EXPECT_TRUE(client->isAlphaNumeric("123456"));
    EXPECT_TRUE(client->isAlphaNumeric("ABCD12"));
    
    EXPECT_FALSE(client->isAlphaNumeric(""));
    EXPECT_FALSE(client->isAlphaNumeric("pass 123")); // space is not alphanumeric
    EXPECT_FALSE(client->isAlphaNumeric("pass!123")); // ! is not alphanumeric
}

TEST_F(BankClientTest, getValidatedString_valid) {
    // 1st case: "john"
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("john"));
    EXPECT_EQ(client->getValidatedString("john"), "john");

    // 2nd case: "  john"
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("  john"));
    EXPECT_EQ(client->getValidatedString("  john"), "john");

    // 3rd case: "john  "
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("john  "));
    EXPECT_EQ(client->getValidatedString("john  "), "john");

    // 4th case: "  john  "
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("  john  "));
    EXPECT_EQ(client->getValidatedString("  john  "), "john");
}

TEST_F(BankClientTest, getValidatedString_invalid) {
    
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(3); 
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(4); // 3 invalid tries + 1 exceeded max tries
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("123"))     // 1st attempt: invalid (has space)
        .WillOnce(testing::Return("123"))     // 2nd attempt: invalid
        .WillOnce(testing::Return("jo hn"));  // 3rd attempt: invalid
        
    auto result = client->getValidatedString("Enter Name");
    auto expected = Result<std::string, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    EXPECT_EQ(result, expected);
}

TEST_F(BankClientTest, getValidatedPassword_valid) {
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("pass"));
    
    EXPECT_EQ(client->getValidatedPassword("Password"), "pass");
}

TEST_F(BankClientTest, getValidatedPassword_alphanumeric) {
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("pass123"));
    
    EXPECT_EQ(client->getValidatedPassword("Password"), "pass123");
}

TEST_F(BankClientTest, getValidatedPassword_invalidLength) {
    // 1st attempt: too long (9 chars)
    // 2nd attempt: quit
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(2);
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("toolongaaaaa"))
        .WillOnce(testing::Return("quit"));

    auto result = client->getValidatedPassword("Password");
    auto expected = Result<std::string, Error::InternalError>::fail(Error::InternalError::USER_CANCELED);
    EXPECT_EQ(result, expected);
}

TEST_F(BankClientTest, getValidatedCurrency_validCases) {
    // Test case insensitivity and mapping
    
    // Case 1: "sgd" -> SGD
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("sgd"));
    EXPECT_EQ(client->getValidatedCurrency("Currency").value(), Protocol::CurrencyType::SGD);

    // Case 2: "USD" -> USD
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("USD"));
    EXPECT_EQ(client->getValidatedCurrency("Currency").value(), Protocol::CurrencyType::USD);

    // Case 3: "Eur" -> EUR
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("Eur"));
    EXPECT_EQ(client->getValidatedCurrency("Currency").value(), Protocol::CurrencyType::EUR);
}

TEST_F(BankClientTest, getValidatedNumber_uint32) {
    // Valid uint32
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("123456"));
    EXPECT_EQ(client->getValidatedNumber<uint32_t>("Account").value(), 123456u);

    // invalid -> valid uint32
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(2);
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("abc")) 
        .WillOnce(testing::Return("123456"));
    EXPECT_EQ(client->getValidatedNumber<uint32_t>("Account").value(), 123456u);
}

TEST_F(BankClientTest, getValidatedNumber_uint16) {
     // Valid double
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("100"));
    EXPECT_EQ(client->getValidatedNumber<uint16_t>("Port").value(), 100u);
   
    // invalid -> valid uint16 
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(2);
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("70000")) // Too large for uint16
        .WillOnce(testing::Return("100"));
    EXPECT_EQ(client->getValidatedNumber<uint16_t>("Port").value(), 100u);
}

TEST_F(BankClientTest, getValidatedNumber_double) {
    // Valid double
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("123.45"));
    EXPECT_DOUBLE_EQ(client->getValidatedNumber<double>("Amount").value(), 123.45);

    // Invalid string -> number
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(2);
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("abc"))
        .WillOnce(testing::Return("50.0"));
    EXPECT_DOUBLE_EQ(client->getValidatedNumber<double>("Amount").value(), 50.0);
}

TEST_F(BankClientTest, fill_account_creation_details_success) {
    Protocol::Command req;
    
    // Expectations for prompt and input
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(4);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("John"))      // name
        .WillOnce(testing::Return("Secret"))    // pw
        .WillOnce(testing::Return("SGD"))       // currency 
        .WillOnce(testing::Return("1000.50"));  // amt

    auto res = client->fill_account_creation_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(req.account_owner_name, "John");
    EXPECT_EQ(req.account_password, "Secret");
    EXPECT_EQ(req.currency, Protocol::CurrencyType::SGD);
    EXPECT_DOUBLE_EQ(req.monetary_value.value(), 1000.50);
}

TEST_F(BankClientTest, fill_auth_details_success) {
    Protocol::Command req;
    
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(3);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Alice"))     // name
        .WillOnce(testing::Return("123456"))    // number (read_line -> stoll)
        .WillOnce(testing::Return("password")); // PW

    auto res = client->fill_auth_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(req.account_owner_name, "Alice");
    EXPECT_EQ(req.account_number, 123456u);
    EXPECT_EQ(req.account_password, "password");
}

TEST_F(BankClientTest, fill_transfer_account_details_success) {
    Protocol::Command req;
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(2);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Bob"))      // TX Name
        .WillOnce(testing::Return("654321"));  // TX Number

    auto res = client->fill_transfer_account_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(req.tx_account_owner_name, "Bob");
    EXPECT_EQ(req.tx_account_number, 654321u);
}

TEST_F(BankClientTest, fill_currency_details_success) {
    Protocol::Command req;
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("EUR"));
    
    auto res = client->fill_currency_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(req.currency, Protocol::CurrencyType::EUR);
}

TEST_F(BankClientTest, fill_amount_details_success) {
    Protocol::Command req;
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("75.25"));
    
    auto res = client->fill_amount_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_DOUBLE_EQ(req.monetary_value.value(), 75.25);
}

TEST_F(BankClientTest, collect_user_input_OPEN) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(1));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(4);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("John"))
        .WillOnce(testing::Return("Pass"))
        .WillOnce(testing::Return("USD"))
        .WillOnce(testing::Return("500"));

    auto res = client->collect_user_input();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::OPEN);
    EXPECT_EQ(res.value().account_owner_name, "John");
}

TEST_F(BankClientTest, collect_user_input_QUIT) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(0));
    
    auto res = client->collect_user_input();
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::USER_QUIT);
}

TEST_F(BankClientTest, SendToServer_Success) {
    Protocol::Command req;
    req.service = Protocol::Service::GET_BALANCE;
    
    std::vector<uint8_t> encoded_cmd = {1, 2, 3};
    std::vector<uint8_t> serialized_msg = {4, 5, 6};
    std::vector<uint8_t> serialized_reply = {7, 8, 9};

    // 1. Mock Encoder: encode_message
    auto mockEncoder = static_cast<MockEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockEncoder, encode_message(testing::_))
        .WillOnce(testing::Return(encoded_cmd));

    // 2. Mock Serializer: serialize request
    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_))
        .WillOnce(testing::Return(serialized_msg));

    // 3. Mock Socket: send and receive
    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    EXPECT_CALL(*mockSocket, send_message(serialized_msg))
        .WillOnce(testing::Return(std::monostate{}));
    EXPECT_CALL(*mockSocket, receive_message())
        .WillOnce(testing::Return(serialized_reply));

    // 4. Mock Serializer: deserialize reply
    Protocol::Message reply_msg;
    reply_msg.type = Protocol::MessageType::Reply;
    reply_msg.payload.status_code = static_cast<uint16_t>(Protocol::ProtocolStatus::SUCCESS);
    reply_msg.payload.content = {10, 11}; // Response content
    EXPECT_CALL(*mockSerializer, deserialize(serialized_reply))
        .WillOnce(testing::Return(reply_msg));

    // 5. Mock Encoder: decode response content
    Protocol::Command res_cmd;
    res_cmd.account_number = 12345;
    res_cmd.monetary_value = 1000.0;
    EXPECT_CALL(*mockEncoder, decode_message(reply_msg.payload.content))
        .WillOnce(testing::Return(res_cmd));

    // Expectations for IO
    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent to server]", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : SUCCESS ]", Colour::GREEN)).Times(1);
    EXPECT_CALL(*mockIO, print_box_top()).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Account Number : 12345"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Balance        : 1000.000000"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print_box_bottom()).Times(1);

    client->send_to_server(req);
}

TEST_F(BankClientTest, SendToServer_ServerError) {
    Protocol::Command req;
    req.service = Protocol::Service::WITHDRAW;
    
    std::vector<uint8_t> serialized_reply = {7, 8, 9};

    auto mockEncoder = static_cast<MockEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockEncoder, encode_message(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{1}));
    
    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{2}));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    EXPECT_CALL(*mockSocket, send_message(testing::_)).WillOnce(testing::Return(std::monostate{}));
    EXPECT_CALL(*mockSocket, receive_message()).WillOnce(testing::Return(serialized_reply));

    Protocol::Message reply_msg;
    reply_msg.type = Protocol::MessageType::Reply;
    reply_msg.payload.status_code = static_cast<uint16_t>(Protocol::ProtocolStatus::INSUFFICIENT_FUNDS);
    EXPECT_CALL(*mockSerializer, deserialize(serialized_reply)).WillOnce(testing::Return(reply_msg));

    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent to server]", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : INSUFFICIENT_FUNDS ]", Colour::RED)).Times(1);

    client->send_to_server(req);
}

TEST_F(BankClientTest, SendToServer_NetworkFailure) {
    Protocol::Command req;
    
    auto mockEncoder = static_cast<MockEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockEncoder, encode_message(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{1}));
    
    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{2}));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    // Fail all send tries (MAX_TRIES is 3)
    EXPECT_CALL(*mockSocket, send_message(testing::_))
        .Times(3)
        .WillRepeatedly(testing::Return(Result<std::monostate, Error::InternalError>::fail(Error::InternalError::SEND_FAILED)));

    EXPECT_CALL(*mockIO, print_error(testing::HasSubstr("Network error"))).Times(3);

    client->send_to_server(req);
}