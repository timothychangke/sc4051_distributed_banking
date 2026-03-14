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

#include "result.h"
#include "internalError.h"

namespace NetworkUtils{
class BaseSocket { 
public:
    BaseSocket(const std::string& ipv4_address, uint16_t port);
    virtual ~BaseSocket();

    virtual Result<std::monostate, Error::InternalError> send_message(const std::vector<uint8_t>& data) = 0;
    virtual Result<std::vector<uint8_t>, Error::InternalError> receive_message() = 0;

protected:
    int sockfd;                // The socket file descriptor
    sockaddr_in address;       // The socket address
    bool isOpen;

};

}