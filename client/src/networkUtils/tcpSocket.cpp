#include "tcpSocket.h"

NetworkUtils::TCPSocket::TCPSocket(const std::string& ipv4_address, uint16_t port) 
    : BaseSocket(ipv4_address, port) 
{
    sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd == -1){
         throw std::runtime_error("[TCPSocket] Invalid sockfd");
    } 
}

NetworkUtils::TCPSocket::~TCPSocket(){}

void NetworkUtils::TCPSocket::connectToServer() {

    if (connect(sockfd,
        (struct sockaddr*)&address,
        sizeof(address)) < 0)
        {
            throw std::runtime_error("[TCPSocket] Connect Failed");
        }

    isOpen = true;
}

void NetworkUtils::TCPSocket::bindToClient() {
    if (bind(sockfd, 
            (struct sockaddr*)&address, 
            sizeof(address)) <0)
        {
            throw std::runtime_error("[TCPSocket] Bind Failed");
        } 
}

bool NetworkUtils::TCPSocket::send_message(const std::vector<uint8_t>& data) {
    // TODO
}

std::vector<uint8_t> NetworkUtils::TCPSocket::receive_message() {
    // TODO
} 