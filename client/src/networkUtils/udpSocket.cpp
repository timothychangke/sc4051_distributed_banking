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

}

std::vector<uint8_t> NetworkUtils::UDPSocket::receive_message() {
    
} 