#pragma once
#include <vector>
#include <cstdint>
#include "protocol.h"
#include "result.h"
#include "internalError.h"

namespace Protocol {

class BaseCommandEncoder {
public:
    virtual ~BaseCommandEncoder() = default;
    
    virtual Result<std::vector<uint8_t>, Error::InternalError> encode_message(const Command& data) = 0;
    virtual Result<Command, Error::InternalError> decode_message(const std::vector<uint8_t>& data) = 0;
};

}