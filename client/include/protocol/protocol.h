#pragma once 
#include <optional>
#include <string>

namespace Protocol{

enum class Service : uint8_t {
    NONE            = 0,
    OPEN            = 1,        
    CLOSE           = 2,       
    DEPOSIT         = 3,             
    WITHDRAW        = 4,            
    MONITOR         = 5,      
    GET_BALANCE     = 6,        
    TRANSFER_FUNDS  = 7,     
};

std::string to_string(Service svc);

enum class CurrencyType {
    SGD,
    USD,
    EUR,
    // add more ...
};

enum class FieldID : uint8_t {
    Service = 1,
    AccountNumber = 2,
    AccountOwnerName = 3,
    AccountPassword = 4,
    TxAccountNumber = 5,
    TxAccountOwnerName = 6,
    MonetaryValue = 7,
    Currency = 8,
    MonitorUpdates = 9,
    MonitorTimeoutSeconds = 10,
};

/** 
* Application Layer
* Represents the business-level operation.
* This structure is serialised and stored inside:
*     Message.payload.content
* The transport layer treats it as raw bytes.
*/
struct Command {
    std::optional<Service> service;

    std::optional<uint32_t> account_number;           // source account 
    std::optional<std::string> account_owner_name;
    std::optional<std::string> account_password;
    
    std::optional<uint32_t> tx_account_number;        // destination account (for transfer)
    std::optional<std::string> tx_account_owner_name;
    
    std::optional<double> monetary_value;
    std::optional<CurrencyType> currency;

    std::optional<std::string> monitor_updates;
    std::optional<uint32_t> monitor_timeout_seconds; 

    auto all_fields() { 
        return std::make_tuple(
            std::make_pair(FieldID::Service, std::ref(service)), 
            std::make_pair(FieldID::AccountNumber, std::ref(account_number)), 
            std::make_pair(FieldID::AccountOwnerName, std::ref(account_owner_name)), 
            std::make_pair(FieldID::AccountPassword, std::ref(account_password)), 
            std::make_pair(FieldID::TxAccountNumber, std::ref(tx_account_number)), 
            std::make_pair(FieldID::TxAccountOwnerName, std::ref(tx_account_owner_name)), 
            std::make_pair(FieldID::MonetaryValue, std::ref(monetary_value)), 
            std::make_pair(FieldID::Currency, std::ref(currency)),
            std::make_pair(FieldID::MonitorUpdates, std::ref(monitor_updates)),
            std::make_pair(FieldID::MonitorTimeoutSeconds, std::ref(monitor_timeout_seconds))
        ); 
    }

    auto all_fields() const { 
        return std::make_tuple(
            std::make_pair(FieldID::Service, std::cref(service)), 
            std::make_pair(FieldID::AccountNumber, std::cref(account_number)), 
            std::make_pair(FieldID::AccountOwnerName, std::cref(account_owner_name)), 
            std::make_pair(FieldID::AccountPassword, std::cref(account_password)), 
            std::make_pair(FieldID::TxAccountNumber, std::cref(tx_account_number)), 
            std::make_pair(FieldID::TxAccountOwnerName, std::cref(tx_account_owner_name)), 
            std::make_pair(FieldID::MonetaryValue, std::cref(monetary_value)), 
            std::make_pair(FieldID::Currency, std::cref(currency)),
            std::make_pair(FieldID::MonitorUpdates, std::cref(monitor_updates)),
            std::make_pair(FieldID::MonitorTimeoutSeconds, std::cref(monitor_timeout_seconds))
        ); 
    }
};

template<typename T, typename Func>
void iterate(T& s, Func f) {
    std::apply([&](auto&... args) {
        ((f(args.first, args.second)), ...);
    }, s.all_fields());
}

}