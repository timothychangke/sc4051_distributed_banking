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
}