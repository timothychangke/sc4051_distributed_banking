#pragma once 

#ifdef _WIN32
    #include <winsock2.h>
    #include <ws2tcpip.h>
#else
    #include <arpa/inet.h>
#endif

#include <vector>
#include <cstdint>
#include <string>
#include <optional>
#include <unordered_map>

#include "protocol.h"
#define FIELD_ID_SIZE 1
#define FIELD_LENGTH 4
#define MAX_STRING_LENGTH 1024

namespace Protocol{
class CommandEncoder{
public:

    CommandEncoder();
    ~CommandEncoder();
    
     /**
     * Converts the Command struct into a packed byte stream.
     * Format: [field_id(1b)][field_length(4b)][field_content(Nb)]
     */
    static std::vector<uint8_t> encode_message(const Command& data);
    
     /**
     * Converts a packed byte stream into the Command struct.
     */
    static std::optional<Command> decode_message(const std::vector<uint8_t>& data);
    
private:

    static size_t get_required_size(const Command& data);       //optimisation 
    static std::optional<FieldID> to_field_id(uint8_t value);

    static void append_uint8(std::vector<uint8_t> &buffer, uint8_t value);
    static void append_uint16(std::vector<uint8_t> &buffer, uint16_t value);
    static void append_uint32(std::vector<uint8_t> &buffer, uint32_t value);
    static void append_double(std::vector<uint8_t> &buffer, double value);
    static void append_string(std::vector<uint8_t>& buffer, const std::string& str);

    static void encode_service(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_account_number(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_account_owner_name(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_account_password(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_tx_account_number(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_tx_account_owner_name(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_monetary_value(std::vector<uint8_t>& buffer, const Command& data);
    static void encode_currency(std::vector<uint8_t>& buffer, const Command& data);
   
    static void decode_service(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_account_password(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_tx_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_tx_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_monetary_value(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static void decode_currency(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);

};
}