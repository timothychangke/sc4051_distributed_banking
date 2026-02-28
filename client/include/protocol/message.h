#pragma once

#include <cstdint>
#include <string>
#include "protocol.h"
#define HEADER_SIZE 17

namespace Protocol{

enum class MessageType: uint8_t {
    Request, 
    Reply,
};

struct MessageId {
    uint32_t request_id;
    uint32_t ipv4_address;
    uint16_t port;
}; 

struct Payload {
    uint16_t status_code; 
    std::string content; 
};

struct Message {
    MessageType type;
    MessageId   id;  // idempotent_id
    Payload     payload;
};

struct Request {
    Service service;
    uint32_t account_number;
    std::string account_owner_name;
    std::string account_password;
    
    uint32_t tx_account_number;
    std::string tx_account_owner_name;
    
    double value;
    CurrencyType currency;
};

}