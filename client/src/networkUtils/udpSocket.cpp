#include <iostream>
#include "udpSocket.h"

NetworkUtils::UDPSocket::UDPSocket(const std::string& ipv4_address, uint16_t port) 
    : BaseSocket(ipv4_address, port) 
{
    sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    if (sockfd == -1){
         throw std::runtime_error("[UDPSocket] Invalid sockfd");
    } 
}

NetworkUtils::UDPSocket::~UDPSocket(){}

bool NetworkUtils::UDPSocket::send_message(const std::vector<uint8_t>& data) {

    if (sendto(
        sockfd,                                      // socket
        reinterpret_cast<const char*>(data.data()),  // message
        data.size(),                                 // length
        0,                                           // flags
        (struct sockaddr*)&address,                  // dest_addr
        sizeof(address)) < 0){                       // dest_len
            return false;
        }
    return true;
}

std::optional<std::vector<uint8_t>> NetworkUtils::UDPSocket::receive_message() {
    
    std::vector<uint8_t> buffer(65535); 
    int bytes_received = recvfrom(
        sockfd, 
        reinterpret_cast<char*>(buffer.data()), 
        static_cast<int>(buffer.size()),        
        0, 
        nullptr, 
        nullptr
    );
    if (bytes_received < 0) {
        return std::nullopt;
    }

    buffer.resize(bytes_received);
    return buffer;
} 