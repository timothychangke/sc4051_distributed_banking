#include "protocolStatus.h"

std::string Protocol::to_string(Protocol::ProtocolStatus status_code) {
    switch (status_code) {
        case ProtocolStatus::SUCCESS:                   return "SUCCESS";
        case ProtocolStatus::ACCOUNT_NOT_FOUND:         return "ACCOUNT_NOT_FOUND";
        case ProtocolStatus::INVALID_CREDENTIALS:       return "INVALID_CREDENTIALS";
        case ProtocolStatus::ACCOUNT_MISMATCH:          return "ACCOUNT_MISMATCH";
        case ProtocolStatus::CURRENCY_MISMATCH:         return "CURRENCY_MISMATCH";
        case ProtocolStatus::INSUFFICIENT_FUNDS:        return "INSUFFICIENT_FUNDS";
        case ProtocolStatus::TRANSFER_SAME_ACCOUNT:     return "TRANSFER_SAME_ACCOUNT";
        case ProtocolStatus::INTERNAL_SERVER_ERROR:     return "INTERNAL_SERVER_ERROR";
        default:                                        return "UNKNOWN_STATUS_CODE";
    }
}