#pragma once

#include <iostream>
#include <string>

enum class Colour { RESET, RED, CYAN, BOLD, BOLD_CYAN };

inline std::string colour_code(Colour c) {
    switch(c) {
        case Colour::RESET: return "\033[0m";
        case Colour::RED: return "\033[31m";
        case Colour::CYAN: return "\033[36m";
        case Colour::BOLD: return "\033[1m";
        case Colour::BOLD_CYAN: return "\033[1;36m";
    }
    return "\033[0m";
}

class BankUI {
public:
    void print(const std::string &msg, Colour colour = Colour::RESET);
    void print_prompt(const std::string &field_name);
    void print_error(const std::string &msg);
    void print_box_top();
    void print_box_bottom();
    void print_service_menu();
};