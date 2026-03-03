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
#include "bankUI.h"
#include "result.h"
#include "internalError.h"

#define MAX_TRIES 3


class BankClient{
public:

    BankClient(std::unique_ptr<BankUI> bankUI);
    ~BankClient();

    void run(); // main loop

private:
    std::unique_ptr<BankUI> bankUI;
    static const std::unordered_map<std::string, Protocol::CurrencyType> stringToCurrency;
     
    Result<Protocol::Command, Error::InternalError> collect_user_input();    
    void send_to_server(const Protocol::Command& req);
    void monitor_server_updates();

    Result<std::monostate, Error::InternalError> fill_account_creation_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_auth_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_currency_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_amount_details(Protocol::Command& req);
    Result<std::monostate, Error::InternalError> fill_transfer_account_details(Protocol::Command& req);
    
    bool BankClient::isValidString(const std::string& str);
    Result<std::string, Error::InternalError> getValidatedString(const std::string& prompt);
    Result<Protocol::CurrencyType, Error::InternalError> getValidatedcurrency(const std::string& prompt);

    template<typename T>
    Result<T, Error::InternalError> getValidatedNumber(const std::string& prompt) {
        static_assert(std::is_arithmetic<T>::value, "T must be numeric");

        std::string input;
        for(int i=0; i < MAX_TRIES; i++) {
            bankUI->print_prompt(prompt + " (or type 'quit' to cancel)");
            std::getline(std::cin, input); 
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
                bankUI->print_error("Invalid " + prompt + " input. Please enter a number");
            } catch (const std::out_of_range&) {
                bankUI->print_error("Invalid " + prompt + " input. Number out of range.");
            }
        }
        bankUI->print_error("Exceeded Maximum Tries");

        return Result<T, Error::InternalError>::fail(
                Error::InternalError::BAD_INPUT);
    }
};