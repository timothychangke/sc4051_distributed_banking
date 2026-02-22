#pragma once 

#include <cstdint>
#include <string>

struct Account {
    std::string account_holder;
    std::string account_id;
    std::string password;
    int32_t account_balance;
};