#pragma once

#include <cstdint>
#include <vector>
#include <string>
#include "baseSocket.h"

namespace NetworkUtils {

class TCPSocket : public BaseSocket {
public:
    TCPSocket(const std::string& ipv4_address, uint16_t port);

    void NetworkUtils::TCPSocket::connectToServer();
    void NetworkUtils::TCPSocket::bindToClient(); 
    
    // High-level message handling
    virtual bool send_message(const std::vector<uint8_t>& data) override;
    virtual std::vector<uint8_t> receive_message() override;

private:
    // Low-level robust helpers
    bool sendAll(const void* data, size_t length);
    bool recvAll(void* buffer, size_t length);
};

}
