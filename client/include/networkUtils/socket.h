#pragma once

#include <cstdint>
#include <vector>
#include <string>

namespace NetworkUtils{
class Socket {
public:
    Socket(uint32_t ipv4_address, uint16_t port);
    ~Socket();

    int connectToServer(); // returns sockfd

    // Sends raw bytes over the connected socket 
    bool sendMessage(const std::vector<char>& data); 
    // Receives raw bytes from the connected socket
    std::vector<char> receiveMessage(); 

private:
    uint32_t ipv4_address;
    uint16_t port;
};

}