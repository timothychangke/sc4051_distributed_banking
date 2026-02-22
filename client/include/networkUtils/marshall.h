#pragma once

#include "message.h"
#include <cstdint>
#include <vector>
#include <string>

namespace NetworkUtils{

class Marshall {
private:
    int sockfd;

public:
    Marshall(int sockfd);
    ~Marshall();

    std::vector<char> serialize(const Message_t& message);
    Message_t deserialize(const std::vector<char>& data);
};

}