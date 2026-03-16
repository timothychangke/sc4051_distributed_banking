#include <iostream>
#include "bankClient.h"

BankClient::BankClient(
    std::unique_ptr<BankIO> bankIO,
    std::unique_ptr<NetworkUtils::BaseSocket> socket,
    std::unique_ptr<Protocol::BaseCommandEncoder> cmdEncoder,
    std::unique_ptr<Protocol::BaseMessageSerializer> msgSerializer
)
    : bankIO(std::move(bankIO)),
      socket(std::move(socket)),
      cmdEncoder(std::move(cmdEncoder)),
      msgSerializer(std::move(msgSerializer)){
    #ifdef _WIN32
        SetConsoleCP(CP_UTF8);
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

    while (true) {
        // std::cout << "\033[2J\033[1;1H"; // Clear screen
        
        bankIO->print_service_menu();
        auto req = collect_user_input();
        if(!req){
            if (req.error() != Error::InternalError::USER_CANCELED){
                Error::InternalError err = req.error();
                bankIO->print_error(Error::to_string(err));
                break;
            }
            else{
                continue;
            }
        }
        bankIO->print("[ SENDING REQUEST TO SERVER ]");
        send_to_server(req.value());
        bankIO->wait_for_enter();

    }
    
}

Result<Protocol::Command, Error::InternalError> BankClient::collect_user_input() {
    int user_input = bankIO->read_int();
    if(user_input == 0){
        return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::USER_QUIT);
    }

    Protocol::Service service_type = static_cast<Protocol::Service>(user_input);
    Protocol::Command req {};
    req.service = service_type;
    
    switch (service_type) {
        case Protocol::Service::OPEN:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_account_creation_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::CLOSE:
        case Protocol::Service::GET_BALANCE:
        case Protocol::Service::MONITOR:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::DEPOSIT:
        case Protocol::Service::WITHDRAW:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::TRANSFER_FUNDS:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_transfer_account_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(req); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        default:
            bankIO->print_error("Invalid Selection");
            return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::INVALID_SERVICE);
    }
    
    return req;
}

void BankClient::trimString(std::string& str) {
    // Trim leading whitespace
    str.erase(str.begin(), std::find_if(str.begin(), str.end(), [](unsigned char ch) {
        return !std::isspace(ch);
    }));

    // Trim trailing whitespace
    str.erase(std::find_if(str.rbegin(), str.rend(), [](unsigned char ch) {
        return !std::isspace(ch);
    }).base(), str.end());
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

bool BankClient::isValidStringLength(const std::string& str) {
    if (str.empty()) return false;
    return str.length() <= MAX_PW_LEN ? true : false;
}

Result<std::string, Error::InternalError> BankClient::getValidatedString(const std::string& prompt){
    for(int i=0; i < MAX_TRIES; i++) {
        bankIO->print_prompt(prompt + " (or type 'quit' to cancel)");
        std::string input = bankIO->read_line(); 

        trimString(input);
        if (input == "quit") {
            return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::USER_CANCELED);
        }

        if(isValidString(input)){
            return input;
        }

        bankIO->print_error("Invalid " + prompt + " string. Try again.");
    }
    bankIO->print_error("Exceeded Maximum Tries");
    
    return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
}

Result<std::string, Error::InternalError> BankClient::getValidatedPassword(const std::string& prompt){
    auto maybe_pw = getValidatedString(prompt);
    if (!maybe_pw) return Result<std::string, Error::InternalError>::fail(maybe_pw.error());
    if (!isValidStringLength(maybe_pw.value())) return Result<std::string, Error::InternalError>::fail(Error::InternalError::BAD_PW_LEN);

    return maybe_pw;
}

Result<Protocol::CurrencyType, Error::InternalError> BankClient::getValidatedCurrency(const std::string& prompt){
    for(int i=0; i < MAX_TRIES; i++) {
        bankIO->print_prompt(prompt + " (or type 'quit' to cancel)");
        std::string input = bankIO->read_line(); 
        if (input == "quit") {
            return Result<Protocol::CurrencyType, Error::InternalError>::fail(
                Error::InternalError::USER_CANCELED);
        }

        for (auto &c : input) c = toupper(c);
        auto it = stringToCurrency.find(input);
        if (it != stringToCurrency.end()) {
            return it->second; 
        }
        bankIO->print_error("Invalid " + prompt + ". Try again.");
    }
    bankIO->print_error("Exceeded Maximum Tries");
    
    return Result<Protocol::CurrencyType, Error::InternalError>::fail(
                Error::InternalError::INVALID_CURRENCY);
}

Result<std::monostate, Error::InternalError> BankClient::fill_account_creation_details(Protocol::Command& req) {
    auto maybe_acc = getValidatedString("Account Holder");
    if (!maybe_acc) return Result<std::monostate, Error::InternalError>::fail(maybe_acc.error());
    req.account_owner_name = maybe_acc.value();

    auto maybe_pwd = getValidatedString("Set Password");
    if (!maybe_pwd) return Result<std::monostate, Error::InternalError>::fail(maybe_pwd.error());
    req.account_password = maybe_pwd.value();

    auto maybe_cur = getValidatedCurrency("Desired currency (SGD/USD/EUR)");
    if (!maybe_cur) return Result<std::monostate, Error::InternalError>::fail(maybe_cur.error());
    req.currency = maybe_cur.value();

    auto maybe_val = getValidatedNumber<double>("Initial Deposit");
    if (!maybe_val) return Result<std::monostate, Error::InternalError>::fail(maybe_val.error());
    req.monetary_value = maybe_val.value();

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_auth_details(Protocol::Command& req) {
    auto maybe_acc_name = getValidatedString("Account Holder");
    if (!maybe_acc_name) return Result<std::monostate, Error::InternalError>::fail(maybe_acc_name.error());
    req.account_owner_name = maybe_acc_name.value();

    auto maybe_acc_num = getValidatedNumber<uint32_t>("Account Number");
    if (!maybe_acc_num) return Result<std::monostate, Error::InternalError>::fail(maybe_acc_num.error());
    req.account_number = maybe_acc_num.value();

    auto maybe_pwd = getValidatedString("Account Password");
    if (!maybe_pwd) return Result<std::monostate, Error::InternalError>::fail(maybe_pwd.error());
    req.account_password = maybe_pwd.value();
    
    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_currency_details(Protocol::Command& req) {
    auto maybe_cur = getValidatedCurrency("Desired currency (SGD/USD/EUR)");
    if (!maybe_cur) return Result<std::monostate, Error::InternalError>::fail(maybe_cur.error());
    req.currency = maybe_cur.value();

   return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_amount_details(Protocol::Command& req) {
    auto maybe_val = getValidatedNumber<double>("Desired Amount");
    if (!maybe_val) return Result<std::monostate, Error::InternalError>::fail(maybe_val.error());
    req.monetary_value = maybe_val.value();

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> BankClient::fill_transfer_account_details(Protocol::Command& req) {
    auto maybe_acc_name = getValidatedString("Transfer Account Holder Name");
    if (!maybe_acc_name) return Result<std::monostate, Error::InternalError>::fail(maybe_acc_name.error());
    req.tx_account_owner_name = maybe_acc_name.value();

    auto maybe_acc_num = getValidatedNumber<uint32_t>("Transfer Account Number");
    if (!maybe_acc_num) return Result<std::monostate, Error::InternalError>::fail(maybe_acc_num.error());
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
