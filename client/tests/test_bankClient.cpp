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

class BankClientTest : public BankClient{
public:
    

};


TEST(InputString, ValidString) {

    BankClientTest client(); 

}