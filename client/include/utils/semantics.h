#pragma once 

#include <random>
#include <cstdint>
#include <unordered_map>

#include "result.h"
#include "internalError.h"

// AT-LEAST-ONCE:
//  - Use timeouts and retransmissions.
//  - Note: Can lead to errors in non-idempotent operations.

// AT-MOST-ONCE:
//  - Use timeouts, Request Identifiers (IDs), and Duplicate Filtering.
//  - Maintain a "Reply History" at the server to re-send lost replies.


namespace Semantics { 
enum class InvocationFlag {
    AT_LEAST_ONCE = 1,
    AT_MOST_ONCE = 2,
};

Result<Semantics::InvocationFlag, Error::InternalError> Semantics::getInvocationFlag(int argc, char* argv[]);
const std::unordered_map<std::string, Semantics::InvocationFlag> Semantics::stringToInvocationFlag;
uint32_t generateRandomUint32(); 
}

