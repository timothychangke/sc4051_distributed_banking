#pragma once

#include <cstdint>
#include <vector>
#include <string>
#include "baseSocket.h"

namespace NetworkUtils{
class UDPSocket : public BaseSocket {
public:
    UDPSocket(const std::string& ipv4_address, uint16_t port);
    virtual ~UDPSocket();

    // UDP does preserve boundaries. One sendto = One recvfrom
    
    // Sends raw bytes over the connected UDPSocket 
    virtual bool send_message(const std::vector<uint8_t>& data) override; 

    // Receives raw bytes from the connected UDPSocket
    virtual std::optional<std::vector<uint8_t>> receive_message() override;
};

}