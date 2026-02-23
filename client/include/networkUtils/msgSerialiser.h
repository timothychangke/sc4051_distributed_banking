#pragma once

#include "message.h"
#include <cstdint>
#include <vector>


namespace NetworkUtils{

class MessageSerializer {
public:
    MessageSerializer(int sockfd);
    ~MessageSerializer();

    /**
     * Converts a Message into a packed byte stream.
     * Format: [Type(4b)][ID(4b)][IP(4b)][Port(2b)][StrLen(4b)][Payload(Nb)]
     */
    std::vector<uint8_t> serialize(const Message& message) const;
    /**
     * Converts a packed byte stream into a Message.
    */
    Message deserialize(const std::vector<uint8_t>& data) const;

private:
    int sockfd;

};

}