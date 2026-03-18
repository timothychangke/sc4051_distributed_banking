#pragma once

#include <iostream>
#include <string>

enum class Colour { RESET, RED, GREEN, YELLOW, CYAN, BOLD, BOLD_CYAN };

inline std::string colour_code(Colour c) {
    switch(c) {
        case Colour::RESET: return "\033[0m";
        case Colour::RED:   return "\033[31m";
        case Colour::GREEN: return "\033[32m";
        case Colour::YELLOW: return "\033[33m";
        case Colour::CYAN:  return "\033[36m";
        case Colour::BOLD:  return "\033[1m";
        case Colour::BOLD_CYAN: return "\033[1;36m";
    }
    return "\033[0m";
}

class BankIO {
public:

    virtual ~BankIO() = default;
    virtual std::string read_line();
    virtual int read_int();
    void ensure_clean_buffer();

    virtual void print(const std::string &msg, Colour colour = Colour::RESET);
    virtual void print_prompt(const std::string &field_name);
    virtual void print_error(const std::string &msg);
    virtual void print_box_top();
    virtual void print_box_bottom();
    virtual void print_service_menu();
    virtual void wait_for_enter();
};