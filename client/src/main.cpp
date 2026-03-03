#include <iostream>
#include <memory>
#include "bankUI.h"
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
    auto bankUI = std::make_unique<BankUI>();
    auto bankClient = std::make_unique<BankClient>(std::move(bankUI)); 
    bankClient->run();
    
    #ifdef _WIN32
    WSACleanup();
    #endif

    return 0 ;
}