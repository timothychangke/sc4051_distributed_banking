#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "bankClient.h"
#include "bankIO.h"
#include "result.h"
#include "internalError.h"

/*

// This test file test for the following functions: 

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
};

class BankClientTestWrapper : public BankClient {
public:
    BankClientTestWrapper(std::unique_ptr<BankIO> io) : BankClient(std::move(io)) {}
    
    // expose methods for testing
    using BankClient::isValidString;
    using BankClient::isValidStringLength;

    using BankClient::getValidatedString;
    using BankClient::getValidatedPassword;
    using BankClient::getValidatedCurrency;
    using BankClient::getValidatedNumber;
    
    using BankClient::fill_account_creation_details;
    using BankClient::fill_auth_details;
    using BankClient::fill_currency_details;
    using BankClient::fill_amount_details;
    using BankClient::fill_transfer_account_details;
};

class BankClientTest : public ::testing::Test {
protected:
    // Pointers to our mock and client. We use pointers so we can control initialization.
    MockBankIO* mockIO;                             // Raw pointer to set expectations
    std::unique_ptr<BankClientTestWrapper> client;  // The object we are testing
    
    void SetUp() override {
    
        auto uniqueMockIO = std::make_unique<MockBankIO>();
        
        // Save a raw pointer to it so we can still set MOCK_EXPECTATIONS on it later,
        // even after we move the unique_ptr into the client.
        mockIO = uniqueMockIO.get();
        
        // Initialize the client wrapper with the mock
        client = std::make_unique<BankClientTestWrapper>(std::move(uniqueMockIO));
    }
   
    void TearDown() override {}
};


TEST_F(BankClientTest, IsValidString_AcceptsValidStrings) {
    EXPECT_TRUE(client->isValidString("John"));
    EXPECT_TRUE(client->isValidString("Doe"));
    EXPECT_TRUE(client->isValidString("ValidName"));
}

TEST_F(BankClientTest, IsValidString_RejectsInvalidStrings) {
    EXPECT_FALSE(client->isValidString(""));  
    EXPECT_FALSE(client->isValidString("John123"));  
    EXPECT_FALSE(client->isValidString("John Doe")); 
    EXPECT_FALSE(client->isValidString("Jane~Doe")); 
}

TEST_F(BankClientTest, IsValidStringLength_ChecksLengthCorrectly) {
    // MAX_PW_LEN is 8
    EXPECT_TRUE(client->isValidStringLength("12345678")); 
    EXPECT_TRUE(client->isValidStringLength("123"));      
    
    EXPECT_FALSE(client->isValidStringLength("123456789")); 
    EXPECT_FALSE(client->isValidStringLength(""));      
}