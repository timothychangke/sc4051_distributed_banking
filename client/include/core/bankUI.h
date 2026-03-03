#include <iostream>
#include <string>

enum class Color { RESET, RED, CYAN, BOLD, BOLD_CYAN };

inline std::string color_code(Color c) {
    switch(c) {
        case Color::RESET: return "\033[0m";
        case Color::RED: return "\033[31m";
        case Color::CYAN: return "\033[36m";
        case Color::BOLD: return "\033[1m";
        case Color::BOLD_CYAN: return "\033[1;36m";
    }
    return "\033[0m";
}

class BankUI {
public:
    void print(const std::string &msg, Color color = Color::RESET);
    void print_prompt(const std::string &field_name);
    void print_error(const std::string &msg);
    void print_box_top();
    void print_box_bottom();
    void print_service_menu();
};