#pragma once 

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
#else
    #include <arpa/inet.h>
#endif

#include <vector>
#include <cstdint>
#include <string>
#include <optional>
#include <unordered_map>
#include <functional>

#include "result.h"
#include "safemath.h"
#include "protocol.h"
#include "internalError.h"
#include "baseCallbackEncoder.h"


namespace Protocol{

class CallbackEncoder : public BaseCallbackEncoder {
public:

    CallbackEncoder();
    ~CallbackEncoder();
    
    /**
     * Converts the CallbackMessage struct into a packed byte stream.
     * Format: [MsgType=0x02(1)][ServiceID(1)][AccountNumber(4)][NameLen(4)][Name(N)][Currency(1)][Balance(8)]
     * Returns ENCODE_EMPTY_COMMAND if no fields are set.
     */
    Result<std::vector<uint8_t>, Error::InternalError> encode_message(const CallbackMessage& data) override;
    
     /**
     * Converts a packed byte stream into the Command struct.
     */
    Result<CallbackMessage, Error::InternalError> decode_message(const std::vector<uint8_t>& data) override;

protected:

    bool validate_payload(size_t total_size); 
};

}
