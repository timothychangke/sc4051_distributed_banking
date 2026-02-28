#pragma once 

#include <vector>
#include <cstdint>
#include <string>

namespace Protocol{
class MessageEncoder{
public:
    MessageEncoder();
    ~MessageEncoder();
    
    static std::vector<uint8_t> encode_message(const std::string& data);
    static std::string decode_message(const std::vector<uint8_t>& data);
    
private:
};
}