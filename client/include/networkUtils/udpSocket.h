#pragma once

#include <cstdint>
#include <vector>
#include <string>
#include "baseSocket.h"

namespace NetworkUtils{
class UDPSocket : public BaseSocket {
public:
    UDPSocket();
    ~UDPSocket();

    // Sends raw bytes over the connected UDPsocket 
    bool send_message(const std::vector<char>& data); 

    // Receives raw bytes from the connected UDPsocket
    std::vector<char> receive_message(); 

private:
    int UDPsocket = -1; 

};

}