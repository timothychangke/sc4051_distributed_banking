#include <iostream>
#include <memory>
#include "bankClient.h"

int main() {

    std::cout << "Hello World" << std::endl; 
    std::unique_ptr<BankClient> bank = std::make_unique<BankClient>();
    bank->run();
    return 0 ;
}