#pragma once 

#include <string>

class errorHandler {
public:
    errorHandler();
    ~errorHandler();

    std::string get_error(int error_code);
};  