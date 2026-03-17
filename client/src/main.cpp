#include <iostream>
#include <memory>

#include "bankIO.h"
#include "udpSocket.h"
#include "msgSerializer.h"
#include "cmdEncoder.h"
#include "bankClient.h"
#include "semantics.h"

int main(int argc, char* argv[]) {

    if (argc < 3) {
        std::cerr << "Usage: " << argv[0] << " <Server_IP> <Server_Port>" << std::endl;
        return 1;
    }

    std::string serverIp = argv[1];
    uint16_t serverPort = static_cast<uint16_t>(std::stoi(argv[2]));
    auto maybe_flag = Semantics::getInvocationFlag(argc, argv);
    if (!maybe_flag) {
        std::cerr << "Warning: Unknown flag. Only '-l' and '-m' " << std::endl;
        return 1;
    }
    Semantics::InvocationFlag flag = maybe_flag.value();

    #ifdef _WIN32
    WSADATA wsaData;
    if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
        throw std::runtime_error("WSAStartup failed");
    }
    #endif

    try {
        // create dependency 
        auto bankIO = std::make_unique<BankIO>();
        auto udpSocket = std::make_unique<NetworkUtils::UDPSocket>(serverIp, serverPort);
        auto cmdEncoder = std::make_unique<Protocol::CommandEncoder>();
        auto msgSerializer = std::make_unique<Protocol::MessageSerializer>();

        // inject dependency :)
        auto bankClient = std::make_unique<BankClient>(
            std::move(bankIO),
            std::move(udpSocket), 
            std::move(cmdEncoder), 
            std::move(msgSerializer),
            flag);
        
        // execute main running loop 
        bankClient->run();
    
    }
    catch (const std::exception& e) {
        std::cerr << "CRITICAL ERROR: " << e.what() << std::endl;
        return 1;
    }
   
    #ifdef _WIN32
    WSACleanup();
    #endif

    return 0 ;
}