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
    For a UDP client, we call connect() to:
    - let the OS assign a local port (implicit bind)
    - select the correct outgoing network interface (local IP)
    - associate a default remote peer (serverAddr)

    This allows us to retrieve the actual local IP/port via getsockname()
    without needing an explicit bind().
    */

    connect_socket();
    local_ip_port = get_local_info();

    #ifdef _WIN32
    DWORD timeout_ms = TIMEOUT * 1000;
    if (setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, (const char*)&timeout_ms, sizeof(timeout_ms)) != 0) {
        throw std::runtime_error("[UDPSocket] setsockopt error");
    }
    #else
    struct timeval timeout;
    timeout.tv_sec = TIMEOUT;
    timeout.tv_usec = 0;
    if (setsockopt(sockfd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout)) < 0) {
        throw std::runtime_error("[UDPSocket] setsockopt error");
    }
    #endif
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

std::pair<uint32_t, uint16_t> NetworkUtils::UDPSocket::get_local_info() {
    sockaddr_in local{};
    socklen_t len = sizeof(local);

    if (getsockname(sockfd, (sockaddr*)&local, &len) < 0) {
        throw std::runtime_error("getsockname failed");
    }

    uint32_t ip = ntohl(local.sin_addr.s_addr); // convert to host order
    uint16_t port = ntohs(local.sin_port);

    return {ip, port};
}