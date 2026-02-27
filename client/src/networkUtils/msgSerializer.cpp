#include "msgSerializer.h"

NetworkUtils::MessageSerializer::MessageSerializer(){};
NetworkUtils::MessageSerializer::~MessageSerializer(){};


/**
 * Converts a Message into a packed byte stream.
 * Format: [Type(1b)][ID(4b)][IP(4b)][Port(2b)][StatusCode(2b)][Content(Nb)]
 */
std::vector<uint8_t> serialize(const NetworkUtils::Message& message)
{
    std::vector<uint8_t> data;

    // m_type (1 Byte)
    data.push_back(static_cast<uint8_t>(message.type));

    // request_id (4 Bytes)
    uint32_t rid = htonl(message.id.request_id);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&rid),
                reinterpret_cast<uint8_t*>(&rid) + 4);

    // IPv4 (4 Bytes)
    uint32_t ip = htonl(message.id.ipv4_address);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&ip),
                reinterpret_cast<uint8_t*>(&ip) + 4);

    // Port (2 Bytes)
    uint16_t port = htons(message.id.port);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&port),
                reinterpret_cast<uint8_t*>(&port) + 2);

    // status_Code (2 Bytes)
    uint16_t sc = htons(message.payload.status_code);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&sc),
                reinterpret_cast<uint8_t*>(&sc) + 2);

    // content length (4 Bytes)
    uint32_t content_len =
        htonl(static_cast<uint32_t>(message.payload.content.size()));

    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&content_len),
                reinterpret_cast<uint8_t*>(&content_len) + 4);

    // content bytes (N Bytes)
    data.insert(data.end(),
                message.payload.content.begin(),
                message.payload.content.end());

    return data;
}


NetworkUtils::Message deserialize(const std::vector<uint8_t>& data){

    NetworkUtils::Message msg{};

    return msg;
}

