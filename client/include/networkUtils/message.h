#pragma once

#include <cstdint>
#include <string>

namespace NetworkUtils{

enum class MessageType {
    Request, 
    Reply,
};

struct MessageId {
    uint32_t idempotent_id;
    uint32_t ipv4_address;
    uint16_t port;
}; 

struct Message {
    MessageType type;
    MessageId   id;
    std::string payload;
};

}