#include "baseSocket.h"

NetworkUtils::BaseSocket::BaseSocket(const std::string& ipv4_address, uint16_t port)
    : sockfd(-1), isOpen(false), address{}
{
    address.sin_family = AF_INET;
    address.sin_port = htons(port);

    if (inet_pton(AF_INET, ipv4_address.c_str(), &address.sin_addr) <= 0) {
        throw std::runtime_error("[BaseSocket] Invalid IPv4 address");
    }
}

NetworkUtils::BaseSocket::~BaseSocket(){
    if (sockfd >= 0) {
        #ifdef _WIN32
            closesocket(sockfd);
        #else
            close(sockfd);
        #endif
        sockfd = -1;
    }
    isOpen = false;
    address = {};
}

