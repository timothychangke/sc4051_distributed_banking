
#ifdef _WIN32
#include <windows.h>
#endif

#pragma once 

#include <cstdint>
#include <string>
#include <optional>
#include <unordered_map>

#include "protocol.h"
#include "message.h"

class BankClient{
public:

    BankClient();
    ~BankClient();

    void run(); // main loop

private:
    static const std::unordered_map<std::string, Protocol::CurrencyType> stringToCurrency;
    
    void print_service_menu();
    void print_top_box();
    void print_bottom_box();
    
    std::optional<Protocol::Request> collect_user_input();
    void fill_auth_details(Protocol::Request& req);
    void fill_currency_details(Protocol::Request& req);
    void fill_amount_details(Protocol::Request& req);
    void fill_transfer_account_details(Protocol::Request& req);
    
    void send_to_server(const Protocol::Request& request);
    void monitor_server_updates();
};