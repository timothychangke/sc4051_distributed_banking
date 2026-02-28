#pragma once

#include <cstdint>
#include <vector>
#include <string>

#include "baseSocket.h"

namespace NetworkUtils {

class TCPSocket : public BaseSocket {
public:
    TCPSocket(const std::string& ipv4_address, uint16_t port);
    virtual ~TCPSocket();

    void connect_to_server();
    void bind_to_client(); 
    
    virtual bool send_message(const std::vector<uint8_t>& data) override;
    virtual std::optional<std::vector<uint8_t>> receive_message() override;

private:
    bool sendAll(const void* data, size_t length);
    bool recvAll(void* buffer, size_t length);
};

}
