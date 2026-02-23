#pragma once

#include <cstdint>
#include <string>

namespace NetworkUtils{

enum class MessageType {
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

}