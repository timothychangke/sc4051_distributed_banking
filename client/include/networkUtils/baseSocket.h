#pragma once

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <sys/socket.h> 
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#endif

#include <stdexcept>
#include <cstdint>
#include <vector>
#include <string>
#include <optional>

namespace NetworkUtils{
class BaseSocket { 
public:
    BaseSocket(const std::string& ipv4_address, uint16_t port);
    virtual ~BaseSocket();

    virtual bool send_message(const std::vector<uint8_t>& data) = 0; 
    virtual std::optional<std::vector<uint8_t>> receive_message() = 0; 

protected:
    int sockfd;                // The socket file descriptor
    sockaddr_in address;       // The socket address
    bool isOpen;

};

}