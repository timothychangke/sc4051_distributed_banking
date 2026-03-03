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
#include <functional>

#include "protocol.h"
#include "helper.h"
#include "result.h"
#include "internalError.h"

#define FIELD_ID_SIZE 1
#define FIELD_LENGTH 4
#define MAX_STRING_LENGTH 1024

namespace Protocol{

using DecoderFunc = std::function<Result<std::monostate, Error::InternalError>(
    Protocol::Command&, size_t&, uint32_t, const std::vector<uint8_t>&)>;

class CommandEncoder{
public:

    CommandEncoder();
    ~CommandEncoder();
    
    /**
     * Converts the Command struct into a packed byte stream.
     * Format: [field_id(1b)][field_length(4b)][field_content(Nb)]
     * Returns ENCODE_EMPTY_COMMAND if no fields are set.
     */
    static Result<std::vector<uint8_t>, Error::InternalError> encode_message(const Command& data);
    
     /**
     * Converts a packed byte stream into the Command struct.
     */
    static Result<Command, Error::InternalError> decode_message(const std::vector<uint8_t>& data);
    
private:
    static const std::unordered_map<FieldID, DecoderFunc> decodeFuncMap;
    
    static size_t get_optimal_buffer_size(const Command& data);       //optimisation 
    static std::optional<FieldID> to_field_id(uint8_t value);
    static bool is_within_data_size(size_t offset,uint32_t length, const std::vector<uint8_t>& data);

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
    
    static Result<std::monostate, Error::InternalError> decode_service(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_account_password(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_tx_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_tx_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_monetary_value(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    static Result<std::monostate, Error::InternalError> decode_currency(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);

};
}