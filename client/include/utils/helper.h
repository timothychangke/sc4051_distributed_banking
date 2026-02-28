#pragma once 

#include <string>

class ErrorHandler {
public:
    ErrorHandler();
    ~ErrorHandler();

    static std::string get_error_msg(int error_code);
    static int get_error_code(std::string error_msg);
};  