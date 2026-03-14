#include <gtest/gtest.h>
#include "bankClient.h"
#include "result.h"
#include "internalError.h"

class BankClientTest : public BankClient{
public:
    using BankClient::isValidString;

};


TEST(InputString, ValidString) {

    BankClientTest client(); 

}

