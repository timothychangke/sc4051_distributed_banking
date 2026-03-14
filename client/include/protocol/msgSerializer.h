#pragma once

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
#else
    #include <arpa/inet.h>
#endif

#include <cstdint>
#include <vector>
#include <optional>

#include "helper.h"
#include "message.h"
#include "result.h"
#include "internalError.h"

namespace Protocol{

class MessageSerializer {
public:
    MessageSerializer();
    ~MessageSerializer();

    /**
     * Converts a Message into a packed byte stream.
     * Format: [Type(4b)][ID(4b)][IP(4b)][Port(2b)][StrLen(4b)][Payload(Nb)]
     */
    Result<std::vector<uint8_t>, Error::InternalError> serialize(const Message& message);
    /**
     * Converts a packed byte stream into a Message.
    */
    Result<Message, Error::InternalError> deserialize(const std::vector<uint8_t>& data);

private:
    bool validate_header(size_t header_size); 
    bool validate_payload(size_t payload_size, size_t offset, uint32_t content_len); 

};

}