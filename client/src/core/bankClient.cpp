#include <iostream>
#include "bankClient.h"

BankClient::BankClient(std::unique_ptr<BankUI> bankUI)
    : bankUI(std::move(bankUI)){
    #ifdef _WIN32
        SetConsoleOutputCP(CP_UTF8);
    #endif
}; 

BankClient::~BankClient(){};

const std::unordered_map<std::string, Protocol::CurrencyType>
BankClient::stringToCurrency = {
    {"SGD", Protocol::CurrencyType::SGD},
    {"USD", Protocol::CurrencyType::USD},
    {"EUR", Protocol::CurrencyType::EUR}
    // more ...
};

void BankClient::run() {

   try {
        while (true) {
            // std::cout << "\033[2J\033[1;1H"; // Clear screen
            bankUI->print_service_menu();
            
            auto req = collect_user_input();
            if(!req){
                if (req.error() != Error::InternalError::USER_CANCELED){
                    Error::InternalError err = req.error();
                    bankUI->print_error(Error::to_string(err));
                }
                break;
            }
        
            bankUI->print("[ SENDING REQUEST TO SERVER ]");
            send_to_server(req.value());
            
        }
    } catch (const std::exception& e) {
        std::cerr << "CRITICAL ERROR: " << e.what() << std::endl;
    }
}

Result<Protocol::Command, Error::InternalError> BankClient::collect_user_input() {
    uint16_t user_input {};
    if(!(std::cin >> user_input) || user_input == 0){
        return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::BAD_INPUT);
    }

    Protocol::Service service_type = static_cast<Protocol::Service>(user_input);
    Protocol::Command req {};
    req.service = service_type;

    switch (service_type) {
        case Protocol::Service::OPEN:
            bankUI->print_box_top();
            if (auto res = fill_account_creation_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankUI->print_box_bottom();
            break;

        case Protocol::Service::CLOSE:
        case Protocol::Service::GET_BALANCE:
        case Protocol::Service::MONITOR:
            bankUI->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankUI->print_box_bottom();
            break;

        case Protocol::Service::DEPOSIT:
        case Protocol::Service::WITHDRAW:
            bankUI->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankUI->print_box_bottom();
            break;

        case Protocol::Service::TRANSFER_FUNDS:
            bankUI->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_transfer_account_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankUI->print_box_bottom();
            break;

        default:
            bankUI->print_error("Invalid Selection");
            return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::INVALID_SERVICE);
    }
    
    return req;
}

bool BankClient::isValidString(const std::string& str) {
    if (str.empty()) return false;

    for (char c : str) {
        if (!std::isalpha(static_cast<unsigned char>(c))) {
            return false;
        }
    }
    return true;
}

Result<std::string, Error::InternalError> BankClient::getValidatedString(const std::string& prompt){
    std::string input;
    for(int i=0; i < MAX_TRIES; i++) {
        bankUI->print_prompt(prompt + " (or type 'quit' to cancel): ");
        std::getline(std::cin, input); 
        if (input == "quit") {
            return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
        }

        if(isValidString(input)){
            return input;
        }

        bankUI->print_error("Invalid" + prompt + "string. Try again.");
    }
    bankUI->print_error("Exceeded Maximum Tries");
    
    return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
}

 Result<Protocol::CurrencyType, Error::InternalError> BankClient::getValidatedcurrency(const std::string& prompt){
    std::string input;
    for(int i=0; i < MAX_TRIES; i++) {
        bankUI->print_prompt(prompt + " (or type 'quit' to cancel): ");
        std::getline(std::cin, input); 
        if (input == "quit") {
            return Result<Protocol::CurrencyType, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
        }

        for (auto &c : input) c = toupper(c);
        auto it = stringToCurrency.find(input);
        if (it != stringToCurrency.end()) {
            return it->second; 
        }
        bankUI->print_error("Invalid" + prompt + ". Try again.");
    }
    bankUI->print_error("Exceeded Maximum Tries");
    
    return Result<Protocol::CurrencyType, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
}

Result<std::monostate, Error::InternalError> BankClient::fill_account_creation_details(Protocol::Command& req) {
    auto maybe_acc = getValidatedString("Account Holder");
    if (maybe_acc.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.account_owner_name = maybe_acc.value();

    auto maybe_pwd = getValidatedString("Set Password");
    if (maybe_pwd.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.account_password = maybe_pwd.value();

    auto maybe_cur = getValidatedcurrency("Desired currency (SGD/USD/EUR)");
    if (maybe_cur.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::INVALID_CURRENCY);
    req.currency = maybe_cur.value();

    auto maybe_val = getValidatedNumber<double>("Initial Deposit");
    if (maybe_val.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.monetary_value = maybe_val.value();

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_auth_details(Protocol::Command& req) {
    auto maybe_acc_name = getValidatedString("Account Holder");
    if (maybe_acc_name.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.account_owner_name = maybe_acc_name.value();

    auto maybe_acc_num = getValidatedNumber<uint32_t>("Account Number");
    if (maybe_acc_num.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.account_number = maybe_acc_num.value();

    auto maybe_pwd = getValidatedString("Account Password");
    if (maybe_pwd.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.account_password = maybe_pwd.value();
    
    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_currency_details(Protocol::Command& req) {
    auto maybe_cur = getValidatedcurrency("Desired currency (SGD/USD/EUR)");
    if (maybe_cur.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::INVALID_CURRENCY);
    req.currency = maybe_cur.value();

   return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_amount_details(Protocol::Command& req) {
    auto maybe_val = getValidatedNumber<double>("Desired Amount");
    if (maybe_val.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.monetary_value = maybe_val.value();

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_transfer_account_details(Protocol::Command& req) {
    auto maybe_acc_name = getValidatedString("Transfer Account Holder Name");
    if (maybe_acc_name.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.tx_account_owner_name = maybe_acc_name.value();

    auto maybe_acc_num = getValidatedNumber<uint32_t>("Transfer Account Number");
    if (maybe_acc_num.fail) return Result<std::monostate, Error::InternalError>::fail(Error::InternalError::BAD_INPUT);
    req.tx_account_number = maybe_acc_num.value();

    return std::monostate{};
}

void BankClient::send_to_server(const Protocol::Command& req) {
    // TODO
    // just need to call functions
}

void BankClient::monitor_server_updates(){
    //TODO
}
