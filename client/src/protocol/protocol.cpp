#include "protocol.h"

std::string Protocol::to_string(Service svc) {
    switch (svc) {
        case Protocol::Service::NONE:            return "SERVICE_NONE";
        case Protocol::Service::OPEN:            return "SERVICE_OPEN"; 
        case Protocol::Service::CLOSE:           return "SERVICE_CLOSE";
        case Protocol::Service::DEPOSIT:         return "SERVICE_DEPOSIT";
        case Protocol::Service::WITHDRAW:        return "SERVICE_WITHDRAW";
        case Protocol::Service::MONITOR:         return "SERVICE_MONITOR";
        case Protocol::Service::GET_BALANCE:     return "SERVICE_GET_BALANCE";
        case Protocol::Service::TRANSFER_FUNDS:  return "SERVICE_TRANSFER_FUNDS";
        
        default:                                 return "UNKNOWN_SERVICE";
    }
}