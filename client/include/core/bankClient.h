#pragma once 

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
    #define NOMINMAX
    #define WIN32_LEAN_AND_MEAN
    #include <windows.h>
#else // _WIN32
    #include <arpa/inet.h>
#endif

#include <cstdint>
#include <vector>
#include <optional>
#include <unordered_map>
#include <memory>
#include <chrono>
#include <thread>
#include <cmath>
#include <algorithm>
#include <cctype> 
#include <limits>
#include <chrono>


#include "protocol.h"
#include "message.h"
#include "bankIO.h"
#include "baseSocket.h"
#include "baseCmdEncoder.h"
#include "baseMsgSerializer.h"

#include "result.h"
#include "internalError.h"
#include "semantics.h"
#include "protocolStatus.h"

#define MAX_TRIES 5
#define MAX_PW_LEN 8
#define BACKOFF 2

class BankClient{
public:

    BankClient( 
        std::unique_ptr<BankIO> bankIO,
        std::unique_ptr<NetworkUtils::BaseSocket> socket,
        std::unique_ptr<Protocol::BaseCommandEncoder> cmdEncoder,
        std::unique_ptr<Protocol::BaseMessageSerializer> msgSerializer,
        Semantics::InvocationFlag flag 
    );
    
    virtual ~BankClient();

    void run(); // main loop

protected:
    std::unique_ptr<BankIO> bankIO;
    std::unique_ptr<NetworkUtils::BaseSocket> socket;
    std::unique_ptr<Protocol::BaseCommandEncoder> cmdEncoder;
    std::unique_ptr<Protocol::BaseMessageSerializer> msgSerializer;
    Semantics::InvocationFlag flag;
    static const std::unordered_map<std::string, Protocol::CurrencyType> stringToCurrency;
    
    Result<Protocol::Message, Error::InternalError> execute_request_pipeline(const Protocol::Command& cmd);
    void execute_client_req(const Protocol::Command& req);
    void monitor_server_updates(const Protocol::Command& cmd);
    void listen_server(uint32_t time);

    Result<std::vector<uint8_t>, Error::InternalError> prepare_message(const Protocol::Command& req);
    Result<std::vector<uint8_t>, Error::InternalError> send_to_server(const std::vector<uint8_t>& data);
    Result<Protocol::Message, Error::InternalError> decode_message(const std::vector<uint8_t>& data);
    Result<std::monostate, Error::InternalError> handle_status_code(const Protocol::Message& msg);
    void decode_command(const Protocol::Message& msg);

    void trimString(std::string& str);
    bool isAlpha(const std::string& str);
    bool isAlphaNumeric(const std::string& str);
    bool isWithinMaxLength(const std::string& str);

    Protocol::Message build_message(const std::vector<uint8_t>& data);
    Result<Protocol::Command, Error::InternalError> build_command();    
    Result<std::monostate, Error::InternalError> fill_account_creation_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_auth_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_currency_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_amount_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_transfer_account_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_monitor_details(Protocol::Command& req);

    Result<std::string, Error::InternalError> getValidatedString(const std::string& prompt);
    Result<std::string, Error::InternalError> getValidatedPassword(const std::string& prompt);
    Result<Protocol::CurrencyType, Error::InternalError> getValidatedCurrency(const std::string& prompt);

    template<typename T>
    Result<T, Error::InternalError> getValidatedNumber(const std::string& prompt) {
        static_assert(std::is_arithmetic<T>::value, "T must be numeric");

        for(int i=0; i < MAX_TRIES; i++) {
            bankIO->print_prompt(prompt + " (or type 'quit' to cancel)");
            std::string input = bankIO->read_line(); 
            trimString(input);
            if (input == "quit") {
                return Result<T, Error::InternalError>::fail(
                    Error::InternalError::USER_CANCELED);
            }

            try {
                if constexpr (std::is_integral<T>::value) {
                    // For integers
                    long long value = std::stoll(input);

                    // Check if T can hold the value
                    if (value < static_cast<long long>((std::numeric_limits<T>::min)()) ||
                        value > static_cast<long long>((std::numeric_limits<T>::max)())) {
                        throw std::out_of_range("Out of range");
                    }

                    if constexpr (std::is_unsigned<T>::value) {
                        if (value < 0) throw std::out_of_range("Unsigned cannot be negative");
                    }

                    return static_cast<T>(value);

                } else if constexpr (std::is_floating_point<T>::value) {
                    // For floating point types
                    double value = std::stod(input);
                    return static_cast<T>(value);
                }

            } catch (const std::invalid_argument&) {
                bankIO->print_error("Invalid " + prompt + " input. Please enter a number");
            } catch (const std::out_of_range&) {
                bankIO->print_error("Invalid " + prompt + " input. Number out of range.");
            }
        }
        bankIO->print_error("Exceeded Maximum Tries");

        return Result<T, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
    }

private:
    uint32_t current_request_id = 0;
};