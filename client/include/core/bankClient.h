#pragma once 

#ifdef _WIN32
#define NOMINMAX
#include <windows.h>
#endif

#include <cstdint>
#include <string>
#include <optional>
#include <unordered_map>
#include <memory>

#include "protocol.h"
#include "message.h"
#include "bankIO.h"
#include "result.h"
#include "internalError.h"

#define MAX_TRIES 3
#define MAX_PW_LEN 8

class BankClient{
public:

    BankClient(std::unique_ptr<BankIO> bankIO);
    ~BankClient();

    void run(); // main loop

protected:
    std::unique_ptr<BankIO> bankIO;
    static const std::unordered_map<std::string, Protocol::CurrencyType> stringToCurrency;
     
    Result<Protocol::Command, Error::InternalError> collect_user_input();    
    void send_to_server(const Protocol::Command& req);
    void monitor_server_updates();

    Result<std::monostate, Error::InternalError> fill_account_creation_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_auth_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_currency_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_amount_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_transfer_account_details(Protocol::Command& req);
    
    void trim(std::string& str);
    bool isValidString(const std::string& str);
    bool isValidStringLength(const std::string& str);

    Result<std::string, Error::InternalError> getValidatedString(const std::string& prompt);
    Result<std::string, Error::InternalError> getValidatedPassword(const std::string& prompt);
    Result<Protocol::CurrencyType, Error::InternalError> getValidatedCurrency(const std::string& prompt);

    template<typename T>
    Result<T, Error::InternalError> getValidatedNumber(const std::string& prompt) {
        static_assert(std::is_arithmetic<T>::value, "T must be numeric");

        for(int i=0; i < MAX_TRIES; i++) {
            bankIO->print_prompt(prompt + " (or type 'quit' to cancel)");
            std::string input = bankIO -> read_line(); 
            if (input == "quit") {
                return Result<T, Error::InternalError>::fail(
                    Error::InternalError::USER_CANCELED);
            }

            try {
                if constexpr (std::is_integral<T>::value) {
                    // For integers
                    long long value = std::stoll(input);

                    // Check if T can hold the value
                    if (value < static_cast<long long>(std::numeric_limits<T>::min()) ||
                        value > static_cast<long long>(std::numeric_limits<T>::max())) {
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
};