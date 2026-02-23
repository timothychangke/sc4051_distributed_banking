#pragma once 

#include <cstdint>
#include <string>
#include <optional>

class BankClient{
public: 
    enum class Service {
        OPEN,        
        CLOSE,       
        DEPOSIT,             
        WITHDRAW,            
        MONITOR,             
        GET_BALANCE,        
        TRANSFER_FUNDS       
    };

    enum class CurrencyType {
        SGD,
        USD,
        EUR,
        // add more ...
    };

    struct Request {
        uint32_t account_number;
        std::string account_owner_name;
        std::string account_password;
        
        uint32_t tx_account_number;
        std::string tx_account_owner_name;
        
        double value;
        CurrencyType currency;
    };

    BankClient();
    ~BankClient();

    void run(); // main loop

private:
    void print_service_menu();
    std::optional<Request> collect_user_input();
    
    bool send_to_server(const Request& request);
    void monitor_server_updates();
};