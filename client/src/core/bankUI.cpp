#include "bankUI.h"

void BankUI::print(const std::string &msg, Color color) {
        std::cout << color_code(color) << msg << color_code(Color::RESET);
    }

void BankUI::print_prompt(const std::string &field_name) {
    print("|  " + field_name + ": ");
}

void BankUI::print_error(const std::string &msg) {
    print("[!] " + msg + "\n", Color::RED);
}

void BankUI::print_box_top() { print("\n" + std::string(31, '─') + "\n", Color::BOLD); }
void BankUI::print_box_bottom() { print(std::string(31, '─') + "\n"); }

void BankUI::print_service_menu() {
    print("\033[1;36m" // Bold Cyan
            "╔══════════════════════════════════════════════════╗\n"
            "║          SC4051 DISTRIBUTED BANK SYSTEM          ║\n"
            "╚══════════════════════════════════════════════════╝\033[0m\n"
            "  [ ACCOUNT ]          [ TRANSACTIONS ]\n"
            "   1. OPEN              3. DEPOSIT\n"
            "   2. CLOSE             4. WITHDRAW\n"
            "                        7. TRANSFER\n\n"
            "  [ INFORMATION ]      [ SYSTEM ]\n"
            "   5. MONITOR           0. EXIT\n"
            "   6. BALANCE\n"
            "────────────────────────────────────────────────────\n"
            " >> Selection: ");
}