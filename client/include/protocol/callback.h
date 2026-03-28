#pragma once

#include <cstdint>
#include <string>

#include "protocol.h"
#include "message.h"

#define MIN_CALLBACK_SIZE 19


namespace Protocol{

struct CallbackMessage {
    MessageType         type;
    Service             service;
    uint32_t            account_number;
    uint32_t            account_owner_name_len ;           
    std::string         account_owner_name;
    CurrencyType        currency;
    uint32_t            monetary_value;
};

}