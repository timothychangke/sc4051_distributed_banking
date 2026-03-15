#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "cmdEncoder.h"
#include "result.h"
#include "internalError.h"
/*
    This test file test for the following functions: 
    -----------------------------------------------

    get_optimal_buffer_size(const Command& data);
    to_field_id(uint8_t value);
    is_within_data_size(size_t offset,uint32_t length, const std::vector<uint8_t>& data);

    append_uint8(std::vector<uint8_t> &buffer, uint8_t value);
    append_uint16(std::vector<uint8_t> &buffer, uint16_t value);
    append_uint32(std::vector<uint8_t> &buffer, uint32_t value);
    append_double(std::vector<uint8_t> &buffer, double value);
    append_string(std::vector<uint8_t>& buffer, const std::string& str);
  
    encode_service(std::vector<uint8_t>& buffer, const Command& data);
    encode_account_number(std::vector<uint8_t>& buffer, const Command& data);
    encode_account_owner_name(std::vector<uint8_t>& buffer, const Command& data);
    encode_account_password(std::vector<uint8_t>& buffer, const Command& data);
    encode_tx_account_number(std::vector<uint8_t>& buffer, const Command& data);
    encode_tx_account_owner_name(std::vector<uint8_t>& buffer, const Command& data);
    encode_monetary_value(std::vector<uint8_t>& buffer, const Command& data);
    encode_currency(std::vector<uint8_t>& buffer, const Command& data);
    
    decode_service(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_account_password(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_tx_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_tx_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_monetary_value(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
    decode_currency(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer);
*/