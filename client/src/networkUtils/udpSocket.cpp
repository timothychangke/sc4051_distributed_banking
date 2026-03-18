#include "udpSocket.h"

NetworkUtils::UDPSocket::UDPSocket(const std::string& ipv4_address, uint16_t port) 
    : BaseSocket(ipv4_address, port) 
{   
    // AF_INET = IPv4, SOCK_DGRAM = UDP
    sockfd = socket(AF_INET, SOCK_DGRAM, 0);
    if (sockfd == -1){
         throw std::runtime_error("[UDPSocket] Invalid sockfd");
    } 

    /*
    given we need the specific IP address rather than the wildcard (0.0.0.0),
    need to bind() the socket to desired local IP before calling sendto().
    */

    connect_socket();
    local_ip_port = get_local_info();
}

NetworkUtils::UDPSocket::~UDPSocket(){}

Result<std::monostate, Error::InternalError>
NetworkUtils::UDPSocket::send_message(const std::vector<uint8_t>& data) {
    if (sendto(
        sockfd,                                      // socket
        reinterpret_cast<const char*>(data.data()),  // message
        data.size(),                                 // length
        0,                                           // flags
        (struct sockaddr*)&address,                  // dest_addr
        sizeof(address)) < 0) {
            return Result<std::monostate, Error::InternalError>::fail(
                Error::InternalError::SEND_FAILED);
        }

    return std::monostate{};
}

Result<std::vector<uint8_t>, Error::InternalError>
NetworkUtils::UDPSocket::receive_message() {
    std::vector<uint8_t> buffer(MAX_DATAGRAM_SIZE); // max datagram size 
    int32_t bytes_received = recvfrom(
        sockfd, 
        reinterpret_cast<char*>(buffer.data()), 
        static_cast<int>(buffer.size()),        
        0, 
        nullptr, 
        nullptr
    );
    if (bytes_received < 0) {
        return Result<std::vector<uint8_t>, Error::InternalError>::fail(
            Error::InternalError::RECEIVE_FAILED);
    }
    buffer.resize(bytes_received);

    return buffer;
} 

Result<std::monostate, Error::InternalError>
NetworkUtils::UDPSocket::bind_socket() {
  
    if(bind(
        sockfd,                         // socket
        (struct sockaddr*)&address,     // dest_addr
        sizeof(address))                // addr_len
        < 0) {
            return Result<std::monostate, Error::InternalError>::fail(
                Error::InternalError::BIND_FAILED);
        }
  
    return std::monostate{};

} 

void NetworkUtils::UDPSocket::connect_socket() {
    if (connect(
        sockfd,
        (struct sockaddr*)&address,
        sizeof(address)
    ) < 0) {
        throw std::runtime_error("getsockname failed");
    }
}

std::pair<std::string, uint16_t> NetworkUtils::UDPSocket::get_local_info() {
    sockaddr_in local{};
    socklen_t len = sizeof(local);

    if (getsockname(sockfd, (sockaddr*)&local, &len) < 0) {
        throw std::runtime_error("getsockname failed");
    }

    char ip[INET_ADDRSTRLEN];
    inet_ntop(AF_INET, &local.sin_addr, ip, sizeof(ip));

    return {ip, ntohs(local.sin_port)};
}