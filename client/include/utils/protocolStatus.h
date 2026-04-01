#pragma once

#include <string>
#include <cstdint>

namespace Protocol{
enum class ProtocolStatus : uint16_t {
    SUCCESS               = 0,
    ACCOUNT_NOT_FOUND     = 1,
    INVALID_CREDENTIALS   = 2, 
    ACCOUNT_MISMATCH      = 3, 
    CURRENCY_MISMATCH     = 4,
    INSUFFICIENT_FUNDS    = 5,
    TRANSFER_SAME_ACCOUNT = 6,
    INTERNAL_SERVER_ERROR = 7,
};

std::string to_string(ProtocolStatus status_code);
}