#include "msgSerializer.h"

NetworkUtils::MessageSerializer::MessageSerializer(){};
NetworkUtils::MessageSerializer::~MessageSerializer(){};


/**
 * Converts a Message into a packed byte stream.
 * Format: [Type(1b)][ID(4b)][IP(4b)][Port(2b)][StatusCode(2b)][ContentLen(4b)][Content(Nb)]
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


std::optional<NetworkUtils::Message> deserialize(const std::vector<uint8_t>& data){

    if (!validate_header(data.size())) {
        return std::nullopt;
    }

    size_t offset = 0;
    NetworkUtils::Message msg{};
    
    // m_type (1 Byte)
    msg.type = static_cast<NetworkUtils::MessageType>(data[offset]);
    offset ++;

    // request_id (4 Bytes)
    uint32_t rid{};
    std::memcpy(&rid, data.data() + offset, 4);
    rid = ntohl(rid);
    msg.id.request_id = rid;
    offset += 4;

    // IPv4 (4 Bytes)
    uint32_t ip{};
    std::memcpy(&ip, data.data() + offset, 4);
    ip = ntohl(ip);
    msg.id.ipv4_address = ip;
    offset += 4;

    // Port (2 Bytes)
    uint16_t port{};
    std::memcpy(&port, data.data() + offset, 2);
    port = ntohs(port);
    msg.id.port = port;
    offset += 2;

    // status_Code (2 Bytes)
    uint16_t sc{};
    std::memcpy(&sc, data.data() + offset, 2);
    sc = ntohs(sc);
    msg.payload.status_code = sc;
    offset += 2;

    // content length (4 Bytes)
    uint32_t content_len{};
    std::memcpy(&content_len, data.data() + offset, 4);
    content_len = ntohl(content_len);
    offset += 4;

    if (!validate_payload(data.size(), offset, content_len)){
        return std::nullopt;
    }

    // content (N Bytes)
    std::string content;
    content.resize(content_len);
    std::memcpy(content.data(), data.data() + offset, content_len);

    return msg;
}

bool validate_header(size_t header_size) {
    if (header_size < HEADER_SIZE)
        return false;
    return true;
}

bool validate_payload(size_t payload_size, size_t offset, uint32_t content_len) {
    if (payload_size < offset + content_len)
        return false;
    return true;
}
