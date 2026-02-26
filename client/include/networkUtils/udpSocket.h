#pragma once

#include <cstdint>
#include <vector>
#include <string>
#include "baseSocket.h"

namespace NetworkUtils{
class UDPSocket : public BaseSocket {
public:
    UDPSocket();
    virtual ~UDPSocket();
    void connect();

    // Sends raw bytes over the connected UDPsocket 
    bool send_message(const std::vector<uint8_t>& data); 

    // Receives raw bytes from the connected UDPsocket
    std::vector<uint8_t> receive_message(); 
};

}