#include "semantics.h"

uint32_t Semantics::generateRandomUint32() {
    static std::random_device rd;
    static std::mt19937 gen(rd()); 
    std::uniform_int_distribution<uint32_t> distr; // Default range is the full type range
    return distr(gen);
}

const std::unordered_map<std::string, Semantics::InvocationFlag>
Semantics::stringToInvocationFlag = {
    {"-l", Semantics::InvocationFlag::AT_LEAST_ONCE},
    {"-m", Semantics::InvocationFlag::AT_MOST_ONCE},
    // more ...
};

Result<Semantics::InvocationFlag, Error::InternalError> Semantics::getInvocationFlag(int argc, char* argv[]){

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];

        auto it = stringToInvocationFlag.find(arg);
        if (it != stringToInvocationFlag.end()) {
            return it->second;
        }
    }
  
    return Result<Semantics::InvocationFlag, Error::InternalError>::fail(
                Error::InternalError::INVALID_INVOCATION_FLAG);
}