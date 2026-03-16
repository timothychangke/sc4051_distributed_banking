#pragma once
#include <vector>
#include <cstdint>
#include "message.h"
#include "result.h"
#include "internalError.h"

namespace Protocol {

class BaseMessageSerializer {
public:
    virtual ~BaseMessageSerializer() = default;

    virtual Result<std::vector<uint8_t>, Error::InternalError> serialize(const Message& message) = 0;
    virtual Result<Message, Error::InternalError> deserialize(const std::vector<uint8_t>& data) = 0;
};

}