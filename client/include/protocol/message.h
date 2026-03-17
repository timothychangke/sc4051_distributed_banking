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
    uint32_t request_id;    // generated on client side 
    uint32_t ipv4_address;
    uint16_t port;
}; 

struct Payload {
    uint16_t status_code; 
    std::string content; 
};

/** 
* Transport Layer
* Represents the network-level packet exchanged between client and server.
* Contains routing/identity metadata and a payload.
* The transport layer does not interpret the payload contents.
*/
struct Message {
    MessageType type;
    MessageId   id;  // idempotent_id
    Payload     payload;
};

}