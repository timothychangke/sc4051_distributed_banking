#pragma once
#include <vector>
#include <cstdint>
#include "callback.h"
#include "result.h"
#include "internalError.h"

namespace Protocol {

class BaseCallbackEncoder {
public:
    virtual ~BaseCallbackEncoder() = default;
    
    virtual Result<std::vector<uint8_t>, Error::InternalError> encode_message(const CallbackMessage& data) = 0;
    virtual Result<CallbackMessage, Error::InternalError> decode_message(const std::vector<uint8_t>& data) = 0;
};

}