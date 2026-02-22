#pragma once

#include <cstdint>
#include <string>

namespace NetworkUtils{

enum class ProtocolType {

    SERVICE_OPEN_ACCOUNT,        // Op1
    SERVICE_CLOSE_ACCOUNT,       // Op2
    SERVICE_DEPOSIT,             // Op3
    SERVICE_WITHDRAW,            // Op4
    SERVICE_MONITOR,             // Op5
    SERVICE_GET_BALANCE,         // Op6: Idempotent
    SERVICE_TRANSFER_FUNDS       // Op7: Non-idempotent
};

enum class MessageType {
    REQUEST,
    REPLY,
};

struct MessageId {
    uint32_t id;
    uint32_t ip4Addr;
    uint16_t port;
}; 

struct Message_t {
    MessageType messageType;
    MessageId messageId;
    std::string content;
};


}