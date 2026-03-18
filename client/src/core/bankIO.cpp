#include "bankIO.h"

void BankIO::print(const std::string &msg, Colour colour) {
        std::cout << colour_code(colour) << msg << colour_code(Colour::RESET);
    }

void BankIO::print_prompt(const std::string &field_name) {
    print("|  " + field_name + ": ");
}

void BankIO::print_error(const std::string &msg) {
    print("[!] " + msg + "\n", Colour::RED);
}

void BankIO::print_box_top()    { print("\n───────────────────────────────\n", Colour::BOLD); }
void BankIO::print_box_bottom() { print("───────────────────────────────\n"); }

void BankIO::print_service_menu() {
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

void BankIO::ensure_clean_buffer() {
    if (std::cin.peek() == '\n') {
        std::cin.ignore();
    }
}

std::string BankIO::read_line() {
    ensure_clean_buffer(); // Make sure a previous '>>' didn't leave a \n
    std::string input;
    if (!std::getline(std::cin, input)) {
        std::cin.clear();
    }
    return input;
}

int BankIO::read_int() {
    int input {};
    if (!(std::cin >> input)) {
        std::cin.clear();
        std::cin.ignore(std::numeric_limits<std::streamsize>::max(), '\n');
    }
    return input;
}

void BankIO::wait_for_enter() {
    std::cout << "Press Enter to continue";
    std::cin.ignore(std::numeric_limits<std::streamsize>::max(), '\n');
    std::cin.get();
}

void BankIO::clear_ui() { 
    std::cout << "\033[2J\033[1;1H";
}