#pragma once 


namespace Protocol{
enum class Service {
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
    Service service;

    uint32_t account_number;           // source account 
    std::string account_owner_name;
    std::string account_password;
    
    uint32_t tx_account_number;        // destination account (for transfer)
    std::string tx_account_owner_name;
    
    double value;
    CurrencyType currency;
};
}