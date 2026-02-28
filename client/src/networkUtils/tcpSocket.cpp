#include "tcpSocket.h"
#include <stdexcept>

NetworkUtils::TCPSocket::TCPSocket(const std::string& ipv4_address, uint16_t port) 
    : BaseSocket(ipv4_address, port) 
{
    sockfd = socket(AF_INET, SOCK_STREAM, 0);
    if (sockfd == -1){
         throw std::runtime_error("[TCPSocket] Invalid sockfd");
    } 
}

NetworkUtils::TCPSocket::~TCPSocket(){}

void NetworkUtils::TCPSocket::connect_to_server() {

    if (connect(sockfd,
        (struct sockaddr*)&address,
        sizeof(address)) < 0)
        {
            throw std::runtime_error("[TCPSocket] Connect Failed");
        }

    isOpen = true;
}

void NetworkUtils::TCPSocket::bind_to_client() {
    if (bind(sockfd, 
            (struct sockaddr*)&address, 
            sizeof(address)) < 0)
        {
            throw std::runtime_error("[TCPSocket] Bind Failed");
        } 
}

bool NetworkUtils::TCPSocket::send_message(const std::vector<uint8_t>& data) {
    int bytes_sent = send(sockfd, reinterpret_cast<const char*>(data.data()), static_cast<int>(data.size()), 0);
    return bytes_sent >= 0;
}

std::optional<std::vector<uint8_t>> NetworkUtils::TCPSocket::receive_message() {
    std::vector<uint8_t> buffer(65535);
    int bytes_received = recv(sockfd, reinterpret_cast<char*>(buffer.data()), static_cast<int>(buffer.size()), 0);
    
    if (bytes_received <= 0) {
        return std::nullopt;
    }

    buffer.resize(bytes_received);
    return buffer;
}