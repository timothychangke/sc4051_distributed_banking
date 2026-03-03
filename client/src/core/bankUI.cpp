#include "bankUI.h"

void BankUI::print(const std::string &msg, Color color = Color::RESET) {
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

// void BankUI::print_service_menu() {
//     std::cout << "\033[1;36m" // Bold Cyan (For fun)
//               << "╔══════════════════════════════════════════════════╗\n"
//               << "║          SC4051 DISTRIBUTED BANK SYSTEM          ║\n"
//               << "╚══════════════════════════════════════════════════╝\033[0m\n"
//               << "  [ ACCOUNT ]          [ TRANSACTIONS ]\n"
//               << "   1. OPEN              3. DEPOSIT\n"
//               << "   2. CLOSE             4. WITHDRAW\n"
//               << "                        7. TRANSFER\n\n"
//               << "  [ INFORMATION ]      [ SYSTEM ]\n"
//               << "   5. MONITOR           0. EXIT\n"
//               << "   6. BALANCE\n"
//               << "────────────────────────────────────────────────────\n"
//               << " >> Selection: ";
// }

// void BankUI::print_top_box() {
//     std::cout << "\n\033[1m┌───────────────────────────────\033[0m\n";
// }

// void BankUI::print_active_service(std::string user_input) {
//     std::cout << "\n\033[1;36m[ ACTIVE SERVICE: " << user_input << " ]\033[0m";
// }

// void BankUI::print_get_account_holder() {
//     std::cout << "│  Account Holder: "; 
// }

// void BankUI::print_invalid_account_holder() {
//     std::cout << "\033[31m[!] Invalid Account Holder Name. Try again.\033[0m\n";
// }

// void BankUI::print_get_account_number() {
//     std::cout << "│  Account Number  : "; 
// }

// void BankUI::print_invalid_account_number() {
//     std::cout << "\033[31m[!] Invalid Account Number. Try again.\033[0m\n";
// }

// void BankUI::print_get_password() {
//     std::cout << "│  Password: ";
// }

// void BankUI::print_set_password() {
//     std::cout << "│  Set Password  : "; 
// }

// void BankUI::print_invalid_password() {
//     std::cout << "\033[31m[!] Invalid Password. Try again.\033[0m\n";
// }

// void BankUI::print_get_currency(){
//     std::cout << "|  Desired currency (SGD/USD/EUR): ";
// }

// void BankUI::print_invalid_currency(){
//     std::cout << "\033[31m[!] Invalid Currency. Try again.\033[0m\n";
// }

// void BankUI::print_get_monetary_amt() {
//     std::cout << "|  Desired Amount : "; 
// }

// void BankUI::print_invalid_monetary_amt() {
//     std::cout << "\033[31m[!] Invalid Amount. Try again.\033[0m\n";
// }

// void BankUI::print_get_transfer_account_holder() {
//     std::cout << "|  Transfer Account Holder Name: "; 
// }

// void BankUI::print_invalid_transfer_account_holder() {
//     std::cout << "\033[31m[!] Invalid Transfer Account Name. Try again.\033[0m\n";
// }

// void BankUI::print_get_transfer_account_number() {
//     std::cout << "|  Transfer Account Number: ";
// }

// void BankUI::print_invalid_transfer_account_number() {
//     std::cout << "\033[31m[!] Invalid Transfer Account Name. Try again.\033[0m\n";
// }

// void BankUI::print_invalid_selection() {
//     std::cout << "\033[31m[!] Invalid Selection. Try again.\033[0m\n";
// }
