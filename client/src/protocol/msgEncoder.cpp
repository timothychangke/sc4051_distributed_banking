#include "msgEncoder.h"


Protocol::MessageEncoder::MessageEncoder(){}
Protocol::MessageEncoder::~MessageEncoder(){}

std::vector<uint8_t> Protocol::MessageEncoder::encode_message(const Protocol::Command& data)
{
    //TODO
}

std::optional<Protocol::Command> Protocol::MessageEncoder::decode_message(const std::vector<uint8_t>& data)
{
    //TODO
}
