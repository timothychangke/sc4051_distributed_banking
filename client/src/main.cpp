#include <iostream>
#include <memory>
#include "bankIO.h"
#include "udpSocket.h"
#include "msgSerializer.h"
#include "cmdEncoder.h"
#include "bankClient.h"

int main(int argc, char* argv[]) {

    std::cout << "Hello World" << std::endl;
    
    #ifdef _WIN32
    WSADATA wsaData;
    if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
        throw std::runtime_error("WSAStartup failed");
    }
    #endif

    // uncomment in production !
    // if (argc < 3) {
    //     std::cerr << "Usage: " << argv[0] << " <IP> <Port>" << std::endl;
    //     return 1;
    // }

    try {
        // create dependency 
        auto bankIO = std::make_unique<BankIO>();
        auto udpSocket = std::make_unique<NetworkUtils::UDPSocket>(argv[1], (uint16_t)std::stoi(argv[2]));
        auto cmdEncoder = std::make_unique<Protocol::CommandEncoder>();
        auto msgSerializer = std::make_unique<Protocol::MessageSerializer>();

        // inject dependency :)
        auto bankClient = std::make_unique<BankClient>(
            std::move(bankIO),
            std::move(udpSocket), 
            std::move(cmdEncoder), 
            std::move(msgSerializer));
        
        // execute main running loop 
        bankClient->run();
    
    }
    catch (const std::exception& e) {
        std::cerr << "CRITICAL ERROR: " << e.what() << std::endl;
    }
   
    #ifdef _WIN32
    WSACleanup();
    #endif

    return 0 ;
}