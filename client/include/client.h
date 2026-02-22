#pragma once
#include "account.h"

class client {

private:

    Account client_account;
    
    int open_account();
    bool close_account();
    bool deposit();
    bool withdraw();
    void monitor_server_updates();
    int get_balance();
    bool transfer_funds();

public:
    client();
    ~client();

    void print_menu();
    bool handle_user_input(int input);
}; 

