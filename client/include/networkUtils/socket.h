#pragma once

#include <cstdint>
#include <vector>
#include <string>

namespace NetworkUtils{

class Socket {
private:

    uint32_t ip4Addr; // IPv4 as 4-byte int
    uint16_t port;

public:
    Socket(uint32_t ip4Addr, uint16_t port);
    ~Socket();

    int connectToServer(); // returns sockfd

    // Sends raw bytes over the connected socket 
    bool sendMessage(const std::vector<char>& data); 
    // Receives raw bytes from the connected socket
    std::vector<char> receiveMessage(); 
};

}