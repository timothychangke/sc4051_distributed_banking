#pragma once

#include <cstdint>
#include <vector>
#include <string>

#include "baseSocket.h"

#define MAX_DATAGRAM_SIZE 65535
#define TIMEOUT 3

namespace NetworkUtils{
class UDPSocket : public BaseSocket {
public:
    UDPSocket(const std::string& ipv4_address, uint16_t port, bool should_connect = true);
    virtual ~UDPSocket();

    virtual Result<std::monostate, Error::InternalError> send_message(const std::vector<uint8_t>& data) override;
    virtual Result<std::vector<uint8_t>, Error::InternalError> receive_message() override;
    virtual Result<std::monostate, Error::InternalError> bind_socket() override;
    virtual std::pair<uint32_t, uint16_t> get_local_info() override;
    void connect_socket();

};

}