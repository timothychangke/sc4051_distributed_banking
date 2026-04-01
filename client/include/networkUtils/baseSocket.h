#pragma once

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#pragma comment(lib, "ws2_32.lib")
#else
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#endif

#include <cstdint>
#include <stdexcept>
#include <cstring>
#include <cerrno>  
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
    virtual Result<std::monostate, Error::InternalError> bind_socket() = 0;
    virtual std::pair<uint32_t, uint16_t> get_local_info() = 0;

    std::pair<uint32_t, uint16_t> local_ip_port;  

protected:
    int sockfd;               // The socket file descriptor
    bool isOpen;
    sockaddr_in address;      // The client/server address | Note this class assume the instance be either a client or server. 
   
};

}