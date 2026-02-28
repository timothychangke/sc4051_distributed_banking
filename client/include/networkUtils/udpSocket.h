#pragma once

#include <cstdint>
#include <vector>
#include <string>

#include "baseSocket.h"
#define MAX_DATAGRAM_SIZE 65535

namespace NetworkUtils{
class UDPSocket : public BaseSocket {
public:
    UDPSocket(const std::string& ipv4_address, uint16_t port);
    virtual ~UDPSocket();

    virtual bool send_message(const std::vector<uint8_t>& data) override; 
    virtual std::optional<std::vector<uint8_t>> receive_message() override;
};

}