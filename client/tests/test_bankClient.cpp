#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "bankClient.h"
#include "bankIO.h"
#include "result.h"
#include "internalError.h"


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

class MockCmdEncoder : public Protocol::BaseCommandEncoder {
public:
    MOCK_METHOD((Result<std::vector<uint8_t>, Error::InternalError>), encode_message, (const Protocol::Command&), (override));
    MOCK_METHOD((Result<Protocol::Command, Error::InternalError>), decode_message, (const std::vector<uint8_t>&), (override));
};

class MockCallbackEncoder : public Protocol::BaseCallbackEncoder {
public:
    MOCK_METHOD((Result<std::vector<uint8_t>, Error::InternalError>), encode_message, (const Protocol::CallbackMessage&), (override));
    MOCK_METHOD((Result<Protocol::CallbackMessage, Error::InternalError>), decode_message, (const std::vector<uint8_t>&), (override));
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
        std::unique_ptr<Protocol::BaseCommandEncoder> cmdEncoder,
        std::unique_ptr<Protocol::BaseCallbackEncoder> callbackEncoder,
        std::unique_ptr<Protocol::BaseMessageSerializer> serializer,
        Semantics::InvocationFlag flag
    ) : BankClient(std::move(io), std::move(socket), std::move(cmdEncoder), std::move(callbackEncoder), std::move(serializer), flag) {}
    
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
    using BankClient::fill_monitor_details;

    using BankClient::build_command;
    using BankClient::execute_client_req;
    using BankClient::monitor_server_updates;
    using BankClient::listen_server;

    Protocol::BaseCommandEncoder* get_encoder() { return cmdEncoder.get(); }
    Protocol::BaseMessageSerializer* get_serializer() { return msgSerializer.get(); }
    NetworkUtils::BaseSocket* get_socket() { return socket.get(); }
    Protocol::BaseCallbackEncoder* get_callback_encoder() { return callbackEncoder.get(); }
};

class BankClientTest : public ::testing::Test {
protected:
    MockBankIO* mockIO;
    std::unique_ptr<BankClientTestWrapper> client;
    
    void SetUp() override {
    
        auto uniqueMockIO = std::make_unique<MockBankIO>();
        auto uniqueMockSocket = std::make_unique<MockSocket>();
        auto uniqueMockCmdEncoder = std::make_unique<MockCmdEncoder>();
        auto uniqueMockCallbackEncoder = std::make_unique<MockCallbackEncoder>();
        auto uniqueMockSerializer = std::make_unique<MockSerializer>();
        
        mockIO = uniqueMockIO.get();
        
        // Initialize the client wrapper with all mocks
        client = std::make_unique<BankClientTestWrapper>(
            std::move(uniqueMockIO),
            std::move(uniqueMockSocket),
            std::move(uniqueMockCmdEncoder),
            std::move(uniqueMockCallbackEncoder),
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
    EXPECT_FALSE(client->isAlphaNumeric("pass 123"));
    EXPECT_FALSE(client->isAlphaNumeric("pass!123"));
}

TEST_F(BankClientTest, getValidatedString_valid) {
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("john"));
    EXPECT_EQ(client->getValidatedString("john"), "john");

    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("  john"));
    EXPECT_EQ(client->getValidatedString("  john"), "john");

    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("john  "));
    EXPECT_EQ(client->getValidatedString("john  "), "john");

    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("  john  "));
    EXPECT_EQ(client->getValidatedString("  john  "), "john");
}

TEST_F(BankClientTest, getValidatedString_invalid) {
    
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(MAX_TRIES); 
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(MAX_TRIES + 1);
    EXPECT_CALL(*mockIO, read_line())
        .Times(MAX_TRIES)
        .WillRepeatedly(testing::Return("123"));
        
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
    
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("sgd"));
    EXPECT_EQ(client->getValidatedCurrency("Currency").value(), Protocol::CurrencyType::SGD);

    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("USD"));
    EXPECT_EQ(client->getValidatedCurrency("Currency").value(), Protocol::CurrencyType::USD);

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

TEST_F(BankClientTest, fill_monitor_details_success) {
    Protocol::Command req;
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("300"));
    
    auto res = client->fill_monitor_details(req);
    EXPECT_TRUE(res.ok());
    EXPECT_EQ(req.monitor_timeout_seconds.value(), 300u);
}

TEST_F(BankClientTest, build_command_OPEN) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(1));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(4);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("John"))
        .WillOnce(testing::Return("Pass"))
        .WillOnce(testing::Return("USD"))
        .WillOnce(testing::Return("500"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::OPEN);
    EXPECT_EQ(res.value().account_owner_name, "John");
}

TEST_F(BankClientTest, build_command_MONITOR) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::MONITOR)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(4);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("John"))
        .WillOnce(testing::Return("123456"))
        .WillOnce(testing::Return("Pass"))
        .WillOnce(testing::Return("300"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::MONITOR);
    EXPECT_EQ(res.value().account_owner_name, "John");
    EXPECT_EQ(res.value().account_number, 123456u);
    EXPECT_EQ(res.value().account_password, "Pass");
    EXPECT_EQ(res.value().monitor_timeout_seconds.value(), 300u);
}

TEST_F(BankClientTest, build_command_QUIT) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(0));
    
    auto res = client->build_command();
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::USER_QUIT);
}


TEST_F(BankClientTest, build_command_CLOSE) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::CLOSE)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(3);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Alice"))
        .WillOnce(testing::Return("123456"))
        .WillOnce(testing::Return("pass"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::CLOSE);
    EXPECT_EQ(res.value().account_owner_name, "Alice");
    EXPECT_EQ(res.value().account_number, 123456u);
    EXPECT_EQ(res.value().account_password, "pass");
}

TEST_F(BankClientTest, build_command_GET_BALANCE) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::GET_BALANCE)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(3);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Bob"))
        .WillOnce(testing::Return("654321"))
        .WillOnce(testing::Return("secret"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::GET_BALANCE);
    EXPECT_EQ(res.value().account_owner_name, "Bob");
    EXPECT_EQ(res.value().account_number, 654321u);
    EXPECT_EQ(res.value().account_password, "secret");
}

TEST_F(BankClientTest, build_command_DEPOSIT) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::DEPOSIT)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(5);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Alice"))
        .WillOnce(testing::Return("111"))
        .WillOnce(testing::Return("pass"))
        .WillOnce(testing::Return("SGD"))
        .WillOnce(testing::Return("200.00"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::DEPOSIT);
    EXPECT_EQ(res.value().account_owner_name, "Alice");
    EXPECT_EQ(res.value().account_number, 111u);
    EXPECT_EQ(res.value().account_password, "pass");
    EXPECT_EQ(res.value().currency, Protocol::CurrencyType::SGD);
    EXPECT_DOUBLE_EQ(res.value().monetary_value.value(), 200.00);
}

TEST_F(BankClientTest, build_command_WITHDRAW) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::WITHDRAW)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(5);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Charlie"))
        .WillOnce(testing::Return("222"))
        .WillOnce(testing::Return("mypass"))
        .WillOnce(testing::Return("USD"))
        .WillOnce(testing::Return("50.00"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::WITHDRAW);
    EXPECT_EQ(res.value().account_owner_name, "Charlie");
    EXPECT_EQ(res.value().account_number, 222u);
    EXPECT_EQ(res.value().account_password, "mypass");
    EXPECT_EQ(res.value().currency, Protocol::CurrencyType::USD);
    EXPECT_DOUBLE_EQ(res.value().monetary_value.value(), 50.00);
}

TEST_F(BankClientTest, build_command_TRANSFER_FUNDS) {
    EXPECT_CALL(*mockIO, read_int()).WillOnce(testing::Return(static_cast<int>(Protocol::Service::TRANSFER_FUNDS)));
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(7);
    EXPECT_CALL(*mockIO, read_line())
        .WillOnce(testing::Return("Alice"))
        .WillOnce(testing::Return("111"))
        .WillOnce(testing::Return("pass"))
        .WillOnce(testing::Return("Bob"))
        .WillOnce(testing::Return("222"))
        .WillOnce(testing::Return("EUR"))
        .WillOnce(testing::Return("100.00"));

    auto res = client->build_command();
    ASSERT_TRUE(res.ok());
    EXPECT_EQ(res.value().service, Protocol::Service::TRANSFER_FUNDS);
    EXPECT_EQ(res.value().account_owner_name, "Alice");
    EXPECT_EQ(res.value().account_number, 111u);
    EXPECT_EQ(res.value().account_password, "pass");
    EXPECT_EQ(res.value().tx_account_owner_name, "Bob");
    EXPECT_EQ(res.value().tx_account_number, 222u);
    EXPECT_EQ(res.value().currency, Protocol::CurrencyType::EUR);
    EXPECT_DOUBLE_EQ(res.value().monetary_value.value(), 100.00);
}

TEST_F(BankClientTest, ExecuteClientReq_Success) {
    Protocol::Command req;
    req.service = Protocol::Service::GET_BALANCE;
    
    std::vector<uint8_t> encoded_cmd = {1, 2, 3};
    std::vector<uint8_t> serialized_msg = {4, 5, 6};
    std::vector<uint8_t> serialized_reply = {7, 8, 9};

    // 1. Mock Encoder: encode_message
    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_))
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
    EXPECT_CALL(*mockCmdEncoder, decode_message(reply_msg.payload.content))
        .WillOnce(testing::Return(res_cmd));

    // Expectations for IO
    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent and received from server]\n", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : SUCCESS ]", Colour::GREEN)).Times(1);
    EXPECT_CALL(*mockIO, print_box_top()).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Account Number   : 12345"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Balance          : 1000.000000"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print_box_bottom()).Times(1);

    client->execute_client_req(req);
}

TEST_F(BankClientTest, ExecuteClientReq_ServerError) {
    Protocol::Command req;
    req.service = Protocol::Service::WITHDRAW;
    
    std::vector<uint8_t> serialized_reply = {7, 8, 9};

    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{1}));
    
    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{2}));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    EXPECT_CALL(*mockSocket, send_message(testing::_)).WillOnce(testing::Return(std::monostate{}));
    EXPECT_CALL(*mockSocket, receive_message()).WillOnce(testing::Return(serialized_reply));

    Protocol::Message reply_msg;
    reply_msg.type = Protocol::MessageType::Reply;
    reply_msg.payload.status_code = static_cast<uint16_t>(Protocol::ProtocolStatus::INSUFFICIENT_FUNDS);
    EXPECT_CALL(*mockSerializer, deserialize(serialized_reply)).WillOnce(testing::Return(reply_msg));

    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent and received from server]\n", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : INSUFFICIENT_FUNDS ]", Colour::RED)).Times(1);

    client->execute_client_req(req);
}

TEST_F(BankClientTest, ExecuteClientReq_NetworkFailure) {
    Protocol::Command req;
    req.service = Protocol::Service::WITHDRAW;
    
    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{1}));
    
    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_)).WillOnce(testing::Return(std::vector<uint8_t>{2}));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    // Fail all send tries
    EXPECT_CALL(*mockSocket, send_message(testing::_))
        .Times(MAX_TRIES)
        .WillRepeatedly(testing::Return(Result<std::monostate, Error::InternalError>::fail(Error::InternalError::SEND_FAILED)));

    EXPECT_CALL(*mockIO, print(testing::HasSubstr("[!] Attempt"), Colour::YELLOW)).Times(MAX_TRIES - 1);
    EXPECT_CALL(*mockIO, print_error(testing::HasSubstr("Final send failure after " + std::to_string(MAX_TRIES) + " attempts: SEND_FAILED"))).Times(1);

    client->execute_client_req(req);
}

TEST_F(BankClientTest, monitor_server_updates_Success) {
    Protocol::Command req;
    req.service = Protocol::Service::MONITOR;
    req.monitor_timeout_seconds = 0; // Don't block listen_server
    
    std::vector<uint8_t> encoded_cmd = {1};
    std::vector<uint8_t> serialized_msg = {2};
    std::vector<uint8_t> serialized_reply = {3};

    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_))
        .WillOnce(testing::Return(encoded_cmd));

    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_))
        .WillOnce(testing::Return(serialized_msg));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    EXPECT_CALL(*mockSocket, send_message(serialized_msg))
        .WillOnce(testing::Return(std::monostate{}));
    
    EXPECT_CALL(*mockSocket, receive_message())
        .WillOnce(testing::Return(serialized_reply));

    Protocol::Message reply_msg;
    reply_msg.type = Protocol::MessageType::Reply;
    reply_msg.payload.status_code = static_cast<uint16_t>(Protocol::ProtocolStatus::SUCCESS);
    reply_msg.payload.content = {4}; 
    
    EXPECT_CALL(*mockSerializer, deserialize(serialized_reply))
        .WillOnce(testing::Return(reply_msg));

    Protocol::Command res_cmd;
    EXPECT_CALL(*mockCmdEncoder, decode_message(reply_msg.payload.content))
        .WillOnce(testing::Return(res_cmd));

    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent and received from server]\n", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : SUCCESS ]", Colour::GREEN)).Times(1);
    EXPECT_CALL(*mockIO, print_box_top()).Times(1);
    EXPECT_CALL(*mockIO, print_box_bottom()).Times(1);
    EXPECT_CALL(*mockIO, print("[ Listening TO SERVER ]\n", Colour::CYAN)).Times(1);

    client->monitor_server_updates(req);
}

TEST_F(BankClientTest, listen_server_TimeoutResilience) {
    auto mockSocket = static_cast<MockSocket*>(client->get_socket());

    EXPECT_CALL(*mockSocket, receive_message())
        .WillRepeatedly(testing::Return(Result<std::vector<uint8_t>, Error::InternalError>::fail(Error::InternalError::RECEIVE_TIMEOUT)));

    client->listen_server(1);
}

TEST_F(BankClientTest, getValidatedCurrency_invalid) {
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(MAX_TRIES);
    EXPECT_CALL(*mockIO, print_error(testing::_)).Times(MAX_TRIES + 1); // MAX_TRIES invalid + 1 exceeded max tries
    EXPECT_CALL(*mockIO, read_line())
        .Times(MAX_TRIES)
        .WillRepeatedly(testing::Return("XYZ"));

    auto result = client->getValidatedCurrency("Currency");
    auto expected = Result<Protocol::CurrencyType, Error::InternalError>::fail(Error::InternalError::INVALID_CURRENCY);
    EXPECT_EQ(result, expected);
}

TEST_F(BankClientTest, getValidatedCurrency_quit) {
    EXPECT_CALL(*mockIO, print_prompt(testing::_)).Times(1);
    EXPECT_CALL(*mockIO, read_line()).WillOnce(testing::Return("quit"));

    auto result = client->getValidatedCurrency("Currency");
    auto expected = Result<Protocol::CurrencyType, Error::InternalError>::fail(Error::InternalError::USER_CANCELED);
    EXPECT_EQ(result, expected);
}

TEST_F(BankClientTest, ExecuteClientReq_RetryThenSucceed) {
    // NOTE: Takes ~2s due to backoff sleep (BACKOFF=2) on the failed first attempt.
    Protocol::Command req;
    req.service = Protocol::Service::GET_BALANCE;

    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_))
        .WillOnce(testing::Return(std::vector<uint8_t>{1, 2, 3}));

    auto mockSerializer = static_cast<MockSerializer*>(client->get_serializer());
    EXPECT_CALL(*mockSerializer, serialize(testing::_))
        .WillOnce(testing::Return(std::vector<uint8_t>{4, 5, 6}));

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    std::vector<uint8_t> serialized_reply = {7, 8, 9};
    EXPECT_CALL(*mockSocket, send_message(testing::_))
        .WillOnce(testing::Return(Result<std::monostate, Error::InternalError>::fail(Error::InternalError::SEND_FAILED)))
        .WillOnce(testing::Return(std::monostate{}));
    EXPECT_CALL(*mockSocket, receive_message())
        .WillOnce(testing::Return(serialized_reply));

    Protocol::Message reply_msg;
    reply_msg.type = Protocol::MessageType::Reply;
    reply_msg.payload.status_code = static_cast<uint16_t>(Protocol::ProtocolStatus::SUCCESS);
    reply_msg.payload.content = {10, 11};
    EXPECT_CALL(*mockSerializer, deserialize(serialized_reply))
        .WillOnce(testing::Return(reply_msg));

    Protocol::Command res_cmd;
    res_cmd.account_number = 99999;
    EXPECT_CALL(*mockCmdEncoder, decode_message(reply_msg.payload.content))
        .WillOnce(testing::Return(res_cmd));

    EXPECT_CALL(*mockIO, print(testing::HasSubstr("[!] Attempt 1 failed"), Colour::YELLOW)).Times(1);
    EXPECT_CALL(*mockIO, print("[SUCCESS: Message sent and received from server]\n", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print("[ SERVER RESPONSE STATUS : SUCCESS ]", Colour::GREEN)).Times(1);
    EXPECT_CALL(*mockIO, print_box_top()).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Account Number   : 99999"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print_box_bottom()).Times(1);

    client->execute_client_req(req);
}

TEST_F(BankClientTest, listen_server_CallbackMessage) {
    std::vector<uint8_t> callback_raw = {
        static_cast<uint8_t>(Protocol::MessageType::Callback), 0x01, 0x02, 0x03
    };

    auto mockSocket = static_cast<MockSocket*>(client->get_socket());
    EXPECT_CALL(*mockSocket, receive_message())
        .WillOnce(testing::Return(callback_raw))
        .WillRepeatedly(testing::Return(
            Result<std::vector<uint8_t>, Error::InternalError>::fail(Error::InternalError::RECEIVE_TIMEOUT)));

    Protocol::CallbackMessage cb_msg;
    cb_msg.type     = Protocol::MessageType::Callback;
    cb_msg.service  = Protocol::Service::DEPOSIT;
    cb_msg.account_number = 12345;
    cb_msg.account_owner_name = "Alice";
    cb_msg.currency = Protocol::CurrencyType::SGD;
    cb_msg.monetary_value = 5000;

    auto mockCallbackEncoder = static_cast<MockCallbackEncoder*>(client->get_callback_encoder());
    EXPECT_CALL(*mockCallbackEncoder, decode_message(callback_raw))
        .WillOnce(testing::Return(cb_msg));

    EXPECT_CALL(*mockIO, print_box_top()).Times(1);
    EXPECT_CALL(*mockIO, print("[ MONITOR CALLBACK UPDATE ]\n", Colour::CYAN)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Service          : SERVICE_DEPOSIT"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Account Number   : 12345"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Account Holder   : Alice"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("Currency         : SGD"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print(testing::HasSubstr("New Balance      : 5000"), testing::_)).Times(1);
    EXPECT_CALL(*mockIO, print_box_bottom()).Times(1);

    client->listen_server(1);
}

TEST_F(BankClientTest, monitor_server_updates_PipelineFailure) {
    Protocol::Command req;
    req.service = Protocol::Service::MONITOR;
    req.monitor_timeout_seconds = 60;

    auto mockCmdEncoder = static_cast<MockCmdEncoder*>(client->get_encoder());
    EXPECT_CALL(*mockCmdEncoder, encode_message(testing::_))
        .WillOnce(testing::Return(
            Result<std::vector<uint8_t>, Error::InternalError>::fail(Error::InternalError::ENCODING_ERROR)));

    EXPECT_CALL(*mockIO, print_error(testing::HasSubstr("Failed to encode command"))).Times(1);

    // listen_server must NOT be called: no socket interactions expected
    client->monitor_server_updates(req);
}