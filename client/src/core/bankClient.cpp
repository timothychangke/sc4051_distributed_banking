#include <iostream>
#include "bankClient.h"

BankClient::BankClient(
    std::unique_ptr<BankIO> bankIO,
    std::unique_ptr<NetworkUtils::BaseSocket> socket,
    std::unique_ptr<Protocol::BaseCommandEncoder> cmdEncoder,
    std::unique_ptr<Protocol::BaseMessageSerializer> msgSerializer,
    Semantics::InvocationFlag flag
)
    : bankIO(std::move(bankIO)),
      socket(std::move(socket)),
      cmdEncoder(std::move(cmdEncoder)),
      msgSerializer(std::move(msgSerializer)),
      flag(flag),
      current_request_id(0){
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
        
        bankIO->clear_ui();
        bankIO->print_service_menu();
        auto req = build_command();
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
        bankIO->print("[ SENDING REQUEST TO SERVER ]\n");
        if (req.value().service == Protocol::Service::MONITOR){
            monitor_server_updates(req.value());
        }   
        else{
            execute_client_req(req.value());
        }
        bankIO->wait_for_enter();

    }
    
}

Result<Protocol::Command, Error::InternalError> BankClient::build_command() {
    int user_input = bankIO->read_int();
    if(user_input == 0){
        return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::USER_QUIT);
    }

    Protocol::Service service_type = static_cast<Protocol::Service>(user_input);
    Protocol::Command cmd {};
    cmd.service = service_type;
    
    switch (service_type) {
        case Protocol::Service::OPEN:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_account_creation_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::CLOSE:
        case Protocol::Service::GET_BALANCE:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::DEPOSIT:
        case Protocol::Service::WITHDRAW:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::TRANSFER_FUNDS:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_transfer_account_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_currency_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            if (auto res = fill_amount_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        case Protocol::Service::MONITOR:
            bankIO->print("ACTIVE SERVICE :" + Protocol::to_string(service_type), Colour::BOLD_CYAN);
            bankIO->print_box_top();
            if (auto res = fill_auth_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
             if (auto res = fill_monitor_details(cmd); !res)
                return Result<Protocol::Command, Error::InternalError>::fail(res.error());
            bankIO->print_box_bottom();
            break;

        default:
            bankIO->print_error("Invalid Selection");
            return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::INVALID_SERVICE);
    }
    
    return cmd;
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

bool BankClient::isAlpha(const std::string& str) {
    if (str.empty()) return false;

    for (char c : str) {
        if (!std::isalpha(static_cast<unsigned char>(c))) {
            return false;
        }
    }
    return true;
}

bool BankClient::isAlphaNumeric(const std::string& str) {
    if (str.empty()) return false;

    return std::all_of(str.begin(), str.end(), [](unsigned char c) {
        return std::isalnum(c); 
    });
}

bool BankClient::isWithinMaxLength(const std::string& str) {
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

        if(isAlpha(input)){
            return input;
        }

        bankIO->print_error("Invalid " + prompt + " string. Try again.");
    }
    bankIO->print_error("Exceeded Maximum Tries");
    
    return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
}

Result<std::string, Error::InternalError> BankClient::getValidatedPassword(const std::string& prompt){
    for(int i=0; i < MAX_TRIES; i++) {
        bankIO->print_prompt(prompt + " (or type 'quit' to cancel)");
        std::string input = bankIO->read_line(); 

        trimString(input);
        if (input == "quit") {
            return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::USER_CANCELED);
        }

        if(isAlphaNumeric(input) && isWithinMaxLength(input)){
            return input;
        }

        bankIO->print_error("Invalid " + prompt + " string. Try again.");
    }
    bankIO->print_error("Exceeded Maximum Tries");
    
    return Result<std::string, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
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

    auto maybe_pwd = getValidatedPassword("Set Password");
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

    auto maybe_pwd = getValidatedPassword("Account Password");
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

Result<std::monostate, Error::InternalError> BankClient::fill_monitor_details(Protocol::Command& req) {
    auto maybe_time = getValidatedNumber<uint32_t>("Desired Monitor Time (In Seconds)");
    if (!maybe_time) return Result<std::monostate, Error::InternalError>::fail(maybe_time.error());
    req.monitor_timeout_seconds = maybe_time.value();

    return std::monostate{};
}

Protocol::Message BankClient::build_message(const std::vector<uint8_t>& data) {

    Protocol::Message msg{};
    msg.type = Protocol::MessageType::Request;
    msg.flag = flag;
    
    if (flag == Semantics::InvocationFlag::AT_MOST_ONCE){
        msg.id.request_id = ++current_request_id; // can use Semantics::generateRandomUint32() for actual prod 
    } else{
        msg.id.request_id = 0; 
    }
    
    auto [local_ip, local_port] = socket->local_ip_port;
    msg.id.ipv4_address = local_ip;
    msg.id.port = local_port;
    msg.payload.status_code = 0;
    msg.payload.content = data;

    return msg;
}

Result<std::vector<uint8_t>, Error::InternalError> BankClient::prepare_message(const Protocol::Command& cmd){

    auto res_enc = cmdEncoder->encode_message(cmd);
    if (!res_enc) {
        bankIO->print_error("[Client] Failed to encode command: " + Error::to_string(res_enc.error()));
        return Result<std::vector<uint8_t>, Error::InternalError>::fail(res_enc.error());
    }
    
    Protocol::Message msg = build_message(res_enc.value());
    auto res_ser = msgSerializer->serialize(msg);
    if (!res_ser) {
        bankIO->print_error("[Client] Failed to serialize message: " + Error::to_string(res_ser.error()));
        return Result<std::vector<uint8_t>, Error::InternalError>::fail(res_ser.error());
    }

    return res_ser;
}

Result<std::vector<uint8_t>, Error::InternalError> BankClient::send_to_server(const std::vector<uint8_t>& data){

    int cur_backoff = BACKOFF;
    Result<std::vector<uint8_t>, Error::InternalError> res_recv;
    Error::InternalError err; 

    for (int i = 1; i <= MAX_TRIES; i++) {
        auto res_send = socket->send_message(data);
        if (res_send) {
            // Loop past any callback packets (0x02) that arrived out-of-order
            // before our expected reply (0x01) — these are monitor push updates
            // buffered in the socket while we were waiting or retrying.
            while (true) {
                res_recv = socket->receive_message();
                if (!res_recv) break; // timeout/error — fall through to retry logic
                const auto& raw = res_recv.value();
                if (!raw.empty() && raw[0] == static_cast<uint8_t>(Protocol::MessageType::Callback)) {
                    continue; // discard stale callback, wait for the real reply
                }
                break; // got a proper reply
            }
        }
        if (res_send && res_recv) {
            bankIO->print("[SUCCESS: Message sent and received from server]\n", Colour::CYAN);
            return res_recv;
        }
        if (i < MAX_TRIES) {
            bankIO->print("[!] Attempt " + std::to_string(i) + " failed. Retrying in " + std::to_string(cur_backoff) + "s...\n", Colour::YELLOW);
            std::this_thread::sleep_for(std::chrono::seconds(cur_backoff));
            cur_backoff *= 2;
        } else {
             if (!res_send) {
                bankIO->print_error("Final send failure after " + std::to_string(MAX_TRIES) + " attempts: " + Error::to_string(res_send.error()));
                err = res_send.error();
            }
             else {
                bankIO->print_error("Final receive failure after " + std::to_string(MAX_TRIES) + " attempts: " + Error::to_string(res_recv.error()));
                err = res_recv.error();
            } 
        }
    }

    return Result<std::vector<uint8_t>, Error::InternalError>::fail(err);
}

Result<Protocol::Message, Error::InternalError> BankClient::decode_message(const std::vector<uint8_t>& data){

    auto res_msg = msgSerializer->deserialize(data);
    if (!res_msg) {
        bankIO->print_error("[Client] Failed to deserialize response: " + Error::to_string(res_msg.error()));
        return Result<Protocol::Message, Error::InternalError>::fail(res_msg.error());
    }
    return res_msg;
}

void BankClient::decode_command(const Protocol::Message& msg){

    if (!msg.payload.content.empty()) {
        
        auto res_cmd_res = cmdEncoder->decode_message(msg.payload.content);
        
        if (!res_cmd_res) {
            bankIO->print_error("[Client] Failed to decode response content: " + Error::to_string(res_cmd_res.error()));
            return;
        }  
        
        const auto& res_cmd = res_cmd_res.value();
        bankIO->print_box_top();
        if (res_cmd.account_number) 
            bankIO->print("Account Number : " + std::to_string(*res_cmd.account_number) + "\n");
        if (res_cmd.monetary_value) 
            bankIO->print("Balance        : " + std::to_string(*res_cmd.monetary_value) + "\n");
        bankIO->print_box_bottom();
    }
}

Result<std::monostate, Error::InternalError> BankClient::handle_status_code(const Protocol::Message& msg){

    Protocol::ProtocolStatus status = static_cast<Protocol::ProtocolStatus>(msg.payload.status_code);
    bankIO->print("[ SERVER RESPONSE STATUS : " + Protocol::to_string(status) + " ]", 
                  status == Protocol::ProtocolStatus::SUCCESS ? Colour::GREEN : Colour::RED);

    if (status != Protocol::ProtocolStatus::SUCCESS) {
        return Result<std::monostate, Error::InternalError>::fail(
                Error::InternalError::BAD_STATUS);
    }

    return std::monostate{};

}

Result<Protocol::Message, Error::InternalError> BankClient::execute_request_pipeline(const Protocol::Command& cmd){

    auto request = prepare_message(cmd);
    if (!request) return Result<Protocol::Message, Error::InternalError>::fail(request.error());
    
    auto response = send_to_server(request.value());
    if (!response) return Result<Protocol::Message, Error::InternalError>::fail(response.error());
    
    auto msg = decode_message(response.value());
    if (!msg) return Result<Protocol::Message, Error::InternalError>::fail(msg.error());

    const auto& res_msg = msg.value();
    auto success = handle_status_code(res_msg);
    if (!success) return Result<Protocol::Message, Error::InternalError>::fail(success.error());

    return res_msg;
}

void BankClient::execute_client_req(const Protocol::Command& cmd) {

    if (cmd.service == Protocol::Service::MONITOR) return; // sanity check
    
    auto res = execute_request_pipeline(cmd);
    if (!res) return;

    decode_command(res.value());
}

void BankClient::listen_server(uint32_t time) {

    auto start = std::chrono::steady_clock::now();
    while (std::chrono::steady_clock::now() - start <
        std::chrono::seconds(static_cast<long long>(time))) {

        auto response = socket->receive_message();
        if (!response){
            if (response.error() != Error::InternalError::RECEIVE_TIMEOUT){
                bankIO->print_error("[Client] Failed to listen to server: " + Error::to_string(response.error()));
            }
            continue;
        }

        const auto& raw = response.value();

        // Debug: dump the full byte stream
        std::string hex;
        hex.reserve(raw.size() * 3);
        char buf[4];
        for (size_t i = 0; i < raw.size(); ++i) {
            std::snprintf(buf, sizeof(buf), "%02X ", raw[i]);
            hex += buf;
        }
        bankIO->print("[DEBUG] listen_server: " + std::to_string(raw.size()) +
                      " bytes received | byte[0]=0x" +
                      (raw.empty() ? "??" : (std::snprintf(buf, sizeof(buf), "%02X", raw[0]), std::string(buf))) +
                      " | " + hex);

        // Route based on first byte: 0x02 = flat callback, anything else = standard reply
        if (!raw.empty() && raw[0] == static_cast<uint8_t>(Protocol::MessageType::Callback)) {
            decode_callback(raw);
            continue;
        }
    }
}

void BankClient::decode_callback(const std::vector<uint8_t>& data) {
    // Server callback wire layout (flat, no standard 18-byte header):
    // [MsgType=0x02(1)][ServiceID(1)][AccountNumber(4)][NameLen(4)][Name(N)][Currency(1)][Balance(8)]
    // Minimum size with empty name: 19 bytes

    constexpr size_t MIN_CALLBACK_SIZE = 1 + 1 + 4 + 4 + 0 + 1 + 8;
    if (data.size() < MIN_CALLBACK_SIZE) {
        bankIO->print_error("[Client] Callback packet too small");
        return;
    }

    size_t offset = 1; // skip MsgType byte already checked

    uint8_t service_id = data[offset++];

    uint32_t account_number{};
    std::memcpy(&account_number, data.data() + offset, 4);
    account_number = ntohl(account_number);
    offset += 4;

    uint32_t name_len{};
    std::memcpy(&name_len, data.data() + offset, 4);
    name_len = ntohl(name_len);
    offset += 4;

    // Bounds check: name + currency(1) + balance(8) must fit
    if (offset + name_len + 1 + 8 > data.size()) {
        bankIO->print_error("[Client] Callback packet malformed");
        return;
    }

    std::string holder_name(reinterpret_cast<const char*>(data.data() + offset), name_len);
    offset += name_len;

    uint8_t currency_raw = data[offset++];

    // Balance: big-endian IEEE 754 float64.
    // Reconstruct as two ntohl'd 32-bit halves — ntohl is already available
    // via winsock2.h / arpa/inet.h pulled in by bankClient.h.
    uint32_t hi{}, lo{};
    std::memcpy(&hi, data.data() + offset,     4);
    std::memcpy(&lo, data.data() + offset + 4, 4);
    uint64_t balance_bits = (static_cast<uint64_t>(ntohl(hi)) << 32) | ntohl(lo);
    double balance{};
    std::memcpy(&balance, &balance_bits, 8);

    // Map server currency byte (1=SGD, 2=USD, 3=EUR) to string
    auto currency_str = [](uint8_t c) -> std::string {
        switch (c) { case 1: return "SGD"; case 2: return "USD"; case 3: return "EUR"; default: return "UNKNOWN"; }
    };

    bankIO->print_box_top();
    bankIO->print("[ MONITOR CALLBACK UPDATE ]");
    bankIO->print("Service          : " + Protocol::to_string(static_cast<Protocol::Service>(service_id)));
    bankIO->print("Account Number   : " + std::to_string(account_number));
    bankIO->print("Account Holder   : " + holder_name);
    bankIO->print("Currency         : " + currency_str(currency_raw));
    bankIO->print("New Balance      : " + std::to_string(balance));
    bankIO->print_box_bottom();
}

void BankClient::monitor_server_updates(const Protocol::Command& cmd) {

    if (cmd.service != Protocol::Service::MONITOR) return; // sanity check

    auto res = execute_request_pipeline(cmd);
    if (!res) return;
    decode_command(res.value());
    bankIO->print("[ Listening TO SERVER ]\n");
    listen_server(cmd.monitor_timeout_seconds.value());
}
