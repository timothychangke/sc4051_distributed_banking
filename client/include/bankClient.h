
#ifdef _WIN32
#include <windows.h>
#endif

#pragma once 

#include <cstdint>
#include <string>
#include <optional>
#include <unordered_map>

class BankClient{
public: 
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

    struct Request {
        Service service;
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
    static const std::unordered_map<std::string, CurrencyType> stringToCurrency;
    
    void print_service_menu();
    void print_top_box();
    void print_bottom_box();
    
    std::optional<Request> collect_user_input();
    void fill_auth_details(Request& req);
    void fill_currency_details(Request& req);
    void fill_amount_details(Request& req);
    void fill_transfer_account_details(Request& req);
    
    void send_to_server(const Request& request);
    void monitor_server_updates();
};