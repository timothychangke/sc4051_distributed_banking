#pragma once

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
#else
#include <sys/socket.h> 
#endif

#include <cstdint>
#include <vector>
#include <string>


namespace NetworkUtils{
class BaseSocket { 
public:
    BaseSocket(uint32_t ipv4_address, uint16_t port);
    virtual ~BaseSocket();

    virtual void create(); 
    virtual void bind(); 

protected:
    int sockfd;                // The socket file descriptor
    sockaddr_in address;       // The socket address
    bool isOpen;

};

}