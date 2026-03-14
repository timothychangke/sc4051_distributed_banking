#include <iostream>
#include "bankClient.h"

const std::unordered_map<std::string, Protocol::CurrencyType>
BankClient::stringToCurrency = {
    {"SGD", Protocol::CurrencyType::SGD},
    {"USD", Protocol::CurrencyType::USD},
    {"EUR", Protocol::CurrencyType::EUR}
    // more ...
};

BankClient::BankClient(){
    #ifdef _WIN32
        SetConsoleOutputCP(CP_UTF8);
    #endif

    // TODO
    // init network and serialiser/encoder logic 


}; 
BankClient::~BankClient(){};

void BankClient::run() {

   try {
        while (true) {
            // std::cout << "\033[2J\033[1;1H"; // Clear screen
            print_service_menu();
            
            auto req = collect_user_input();
            if (!req) break; // Exit on 0 or invalid

            
            std::cout << "\n\033[33m[ SENDING REQUEST TO SERVER... ]\033[0m\n";
            send_to_server(req.value());
            
        }
    } catch (const std::exception& e) {
        std::cerr << "CRITICAL ERROR: " << e.what() << std::endl;
        // Attempt recovery or safe shutdown
    }

}

void BankClient::print_service_menu() {
    std::cout << "\033[1;36m" // Bold Cyan (For fun)
              << "╔══════════════════════════════════════════════════╗\n"
              << "║          SC4051 DISTRIBUTED BANK SYSTEM          ║\n"
              << "╚══════════════════════════════════════════════════╝\033[0m\n"
              << "  [ ACCOUNT ]          [ TRANSACTIONS ]\n"
              << "   1. OPEN              3. DEPOSIT\n"
              << "   2. CLOSE             4. WITHDRAW\n"
              << "                        7. TRANSFER\n\n"
              << "  [ INFORMATION ]      [ SYSTEM ]\n"
              << "   5. MONITOR           0. EXIT\n"
              << "   6. BALANCE\n"
              << "────────────────────────────────────────────────────\n"
              << " >> Selection: ";
}

void BankClient::print_top_box() {
    std::cout << "\n\033[1m┌───────────────────────────────\033[0m\n";
}

void BankClient::print_bottom_box() {
        std::cout << " └───────────────────────────────┘\n";
}

std::optional<Protocol::Command> BankClient::collect_user_input() {
    uint16_t user_input {};
    if(!(std::cin >> user_input) || user_input == 0) return std::nullopt;

    std::cout << "\n\033[1;36m[ ACTIVE SERVICE: " << user_input << " ]\033[0m";
    Protocol::Service service_type = static_cast<Protocol::Service>(user_input);

    Protocol::Command req {};
    req.service = service_type;
    switch (service_type) {
        case Protocol::Service::OPEN:
            print_top_box();
            std::cout << "  │  Service: OPEN ACCOUNT\n";
            std::cout << "  │  Account Holder: "; 
            std::getline(std::cin >> std::ws, req.account_owner_name.value());
            std::cout << "  │  Set Password  : "; 
            std::cin >> req.account_password.value();
            fill_currency_details(req);
            fill_amount_details(req);
            print_bottom_box();
            break;

        case Protocol::Service::CLOSE:
        case Protocol::Service::GET_BALANCE:
        case Protocol::Service::MONITOR:
            print_top_box();
            fill_auth_details(req);
            print_bottom_box();
            break;
        
        case Protocol::Service::DEPOSIT:
        case Protocol::Service::WITHDRAW:
            print_top_box();
            fill_auth_details(req);
            fill_currency_details(req);
            fill_amount_details(req);
            print_bottom_box();
            break;

        case Protocol::Service::TRANSFER_FUNDS:
            print_top_box();
            fill_auth_details(req);
            fill_transfer_account_details(req);
            fill_currency_details(req);
            fill_amount_details(req);
            print_bottom_box();
            break;
        
        default:
            std::cout << "\033[31mInvalid Selection\033[0m\n";
            return std::nullopt;
    }
    
    return req;
}

void BankClient::fill_auth_details(Protocol::Command& req) {
    std::cout << "│  Account Holder  : "; std::getline(std::cin >> std::ws, req.account_owner_name.value());
    std::cout << "│  Account Number  : "; std::cin >> req.account_number.value();
    std::cout << "│  Password: "; std::cin >> req.account_password.value();
}

void BankClient::fill_currency_details(Protocol::Command& req) {
    std::string user_input {};
    while (true) {
        std::cout << "|  Desired currency (SGD/USD/EUR): "; std::cin >> user_input;
        
        for (auto &c : user_input) c = toupper(c);

        auto it = stringToCurrency.find(user_input);
        if (it != stringToCurrency.end()) {
            req.currency = it->second;
            break; 
        }
        std::cout << "\033[31m[!] Invalid Currency. Try again.\033[0m\n";
    }
}

void BankClient::fill_amount_details(Protocol::Command& req) {
    std::cout << "|  Desired Amount : "; std::cin >> req.monetary_value.value(); 
}

void BankClient::fill_transfer_account_details(Protocol::Command& req) {
    std::cout << "|  Transfer Account Holder Name: "; std::getline(std::cin >> std::ws, req.tx_account_owner_name.value());
    std::cout << "|  Transfer Account Number: "; std::cin >> req.tx_account_number.value();
}

void BankClient::send_to_server(const Protocol::Command& req) {
    // TODO
    // just need to call functions
}

void BankClient::monitor_server_updates(){
    //TODO
}
