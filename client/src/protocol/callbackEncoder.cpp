#include "callbackEncoder.h"

Protocol::CallbackEncoder::CallbackEncoder(){};
Protocol::CallbackEncoder::~CallbackEncoder(){};

Result<std::vector<uint8_t>, Error::InternalError>
Protocol::CallbackEncoder::encode_message(const Protocol::CallbackMessage& cb_message){
    std::vector<uint8_t> data;

    // cb_type (1 Byte)
    data.push_back(static_cast<uint8_t>(cb_message.type));

    // cb_service_id (1 Byte)
    data.push_back(static_cast<uint8_t>(cb_message.service));
    
    // acc_num (4 Bytes)
    uint32_t acc_num = htonl(cb_message.account_number);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&acc_num),
                reinterpret_cast<uint8_t*>(&acc_num) + 4);

    // acc_holder_name_len (4 Bytes)
    uint32_t acc_name_len = htonl(cb_message.account_owner_name_len);
    data.insert(data.end(),
                reinterpret_cast<uint8_t*>(&acc_name_len),
                reinterpret_cast<uint8_t*>(&acc_name_len) + 4);

    // acc_holder_name (N Bytes)
    data.insert(data.end(),
                cb_message.account_owner_name.begin(),
                cb_message.account_owner_name.end());

    // currency (1 Byte)
    data.push_back(static_cast<uint8_t>(cb_message.currency));

    // monetary_value (8 Bytes)
    uint64_t val;
    std::memcpy(&val, &cb_message.monetary_value, sizeof(uint64_t));
    val = ((val & 0xFF00000000000000ULL) >> 56) |
        ((val & 0x00FF000000000000ULL) >> 40) |
        ((val & 0x0000FF0000000000ULL) >> 24) |
        ((val & 0x000000FF00000000ULL) >> 8)  |
        ((val & 0x00000000FF000000ULL) << 8)  |
        ((val & 0x0000000000FF0000ULL) << 24) |
        ((val & 0x000000000000FF00ULL) << 40) |
        ((val & 0x00000000000000FFULL) << 56);
    uint8_t* ptr = reinterpret_cast<uint8_t*>(&val);
    data.insert(
        data.end(),
        ptr,
        ptr + sizeof(uint64_t));

    return data;
}

Result<Protocol::CallbackMessage, Error::InternalError>
Protocol::CallbackEncoder::decode_message(const std::vector<uint8_t>& data){
    
    if (!validate_payload(data.size()))
    return Result<Protocol::CallbackMessage, Error::InternalError>::fail(
        Error::InternalError::DESERIALIZE_PAYLOAD_OVERFLOW);

    size_t offset = 0;
    Protocol::CallbackMessage cb_msg{};
    
    // cb_type (1 Byte)
    cb_msg.type = static_cast<Protocol::MessageType>(data[offset]);
    offset ++;
    
    // cb_service_id (1 Byte)
    cb_msg.service = static_cast<Protocol::Service>(data[offset]);
    offset ++;

    // acc_num (4 Bytes)
    uint32_t acc_num{};
    std::memcpy(&acc_num, data.data() + offset, 4);
    acc_num = ntohl(acc_num);
    cb_msg.account_number = acc_num;
    offset += 4;

    // acc_holder_name_len (4 Bytes)
    uint32_t acc_name_len{};
    std::memcpy(&acc_name_len, data.data() + offset, 4);
    acc_name_len = ntohl(acc_name_len);
    cb_msg.account_owner_name_len = acc_name_len;
    offset += 4;

    // acc_name (N Bytes)
    cb_msg.account_owner_name.resize(acc_name_len);
    std::memcpy(cb_msg.account_owner_name.data(),
                data.data() + offset,
                acc_name_len);
    offset += acc_name_len;

    // currency (1 Byte)
    cb_msg.currency= static_cast<Protocol::CurrencyType>(data[offset]);
    offset ++;

    // monetary_value (8 Bytes)
    uint64_t val;
    std::memcpy(&val, data.data() + offset, 8);

    // manual "ntohll" 
    val = ((val & 0xFF00000000000000ULL) >> 56) |
          ((val & 0x00FF000000000000ULL) >> 40) |
          ((val & 0x0000FF0000000000ULL) >> 24) |
          ((val & 0x000000FF00000000ULL) >> 8)  |
          ((val & 0x00000000FF000000ULL) << 8)  |
          ((val & 0x0000000000FF0000ULL) << 24) |
          ((val & 0x000000000000FF00ULL) << 40) |
          ((val & 0x00000000000000FFULL) << 56);

    double mon_val;
    std::memcpy(&mon_val, &val, sizeof(double));
    cb_msg.monetary_value = mon_val;

    return cb_msg;
}

bool Protocol::CallbackEncoder::validate_payload(size_t total_size) {    
    if (total_size < MIN_CALLBACK_SIZE) return false;

    return true;
}
