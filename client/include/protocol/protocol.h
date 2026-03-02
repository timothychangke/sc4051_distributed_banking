#pragma once 
#include <optional>

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

enum class CurrencyType {
    SGD,
    USD,
    EUR,
    // add more ...
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
};

}