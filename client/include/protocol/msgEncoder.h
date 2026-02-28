#pragma once 

#include <vector>
#include <cstdint>
#include <string>

namespace Protocol{
class MessageEncoder{
public:
    MessageEncoder();
    ~MessageEncoder();
    
     /**
     * Converts a Message into a packed byte stream.
     * Format: [Type(4b)][ID(4b)][IP(4b)][Port(2b)][StrLen(4b)][Payload(Nb)]
     */
    static std::vector<uint8_t> encode_message(const std::string& data);
    
     /**
     * Converts a Message into a packed byte stream.
     * Format: [Type(4b)][ID(4b)][IP(4b)][Port(2b)][StrLen(4b)][Payload(Nb)]
     */
    static std::string decode_message(const std::vector<uint8_t>& data);
    
private:
};
}