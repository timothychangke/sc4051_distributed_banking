#include <iostream>
#include <memory>
#include "bankClient.h"

int main() {

    std::cout << "Hello World" << std::endl;
    
    #ifdef _WIN32
    WSADATA wsaData;
    if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
        throw std::runtime_error("WSAStartup failed");
    }
    #endif

    // Core Logic
    std::unique_ptr<BankClient> bank = std::make_unique<BankClient>();
    bank->run();
    
    
    #ifdef _WIN32
    WSACleanup();
    #endif

    return 0 ;
}