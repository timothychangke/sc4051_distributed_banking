#include "cmdEncoder.h"


Protocol::CommandEncoder::CommandEncoder(){}
Protocol::CommandEncoder::~CommandEncoder(){}

Result<std::vector<uint8_t>, Error::InternalError>
Protocol::CommandEncoder::encode_message(const Protocol::Command& data){
    
    std::vector<uint8_t> buffer {};
    buffer.reserve(CommandEncoder::get_optimal_buffer_size(data));

    std::optional<Error::InternalError> error {};
    iterate(data, [&](auto fieldId, const auto& field) {
        if (error || !field.has_value()) return;

        auto it = encodeFuncMap.find(fieldId);
        if (it == encodeFuncMap.end()) {
            error = Error::InternalError::ENCODE_UNKNOWN_FIELD;
            return;
        }

        auto res = it->second(buffer, data);
        if (!res) {
            error = res.error();
            return;
        }
    });

    if (error) {
        return Result<std::vector<uint8_t>, Error::InternalError>::fail(*error);
    }

    if (buffer.empty()) {
        return Result<std::vector<uint8_t>, Error::InternalError>::fail(
            Error::InternalError::ENCODE_EMPTY_COMMAND);
    }

    return buffer;
}

Result<Protocol::Command, Error::InternalError>
Protocol::CommandEncoder::decode_message(const std::vector<uint8_t>& data){
    
    if (data.empty())
        return Result<Protocol::Command, Error::InternalError>::fail(
            Error::InternalError::DECODE_EMPTY_DATA);

    Protocol::Command cmd{};
    size_t offset {0};
    while (true) {
        auto sum1 = Safe_math::safe_add(offset, FIELD_ID_SIZE);
        if (!sum1)
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_OFFSET_OVERFLOW);

        auto sum2 = Safe_math::safe_add(*sum1, FIELD_LENGTH);
        if (!sum2)
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_OFFSET_OVERFLOW);

        if (*sum2 > data.size()) break;
        
        // decode [field_id(1b)]
        uint8_t field_id = data[offset];
        std::optional<Protocol::FieldID> field = CommandEncoder::to_field_id(field_id);
        if (!field.has_value())
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_UNKNOWN_FIELD);
        
        auto maybe_offset = Safe_math::safe_add(offset, FIELD_ID_SIZE);
        if (!maybe_offset)
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_OFFSET_OVERFLOW);
        offset = *maybe_offset;

        // decode [field_length(4b)]
        uint32_t length {};
        std::memcpy(&length, &data[offset], sizeof(uint32_t));
        length = ntohl(length); 
        
        maybe_offset = Safe_math::safe_add(offset, FIELD_LENGTH);
        if (!maybe_offset)
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_OFFSET_OVERFLOW);
        offset = *maybe_offset;

        if(!is_within_data_size(offset, length, data))
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_FIELD_OVERFLOW);
        
        // decode [field_content(Nb)]
        auto it = decodeFuncMap.find(field.value());
        if (it == decodeFuncMap.end()) {
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_UNKNOWN_FIELD);
        }

        auto res = it->second(cmd, offset, length, data);
        if (!res) {
            return Result<Protocol::Command, Error::InternalError>::fail(res.error());
        }

        auto next_offset = Safe_math::safe_add(offset, length);
        if (!next_offset)
            return Result<Protocol::Command, Error::InternalError>::fail(
                Error::InternalError::DECODE_OFFSET_OVERFLOW);
        offset = *next_offset;
    }

    return cmd;
}

// note: this optimisation not really required
size_t Protocol::CommandEncoder::get_optimal_buffer_size(const Protocol::Command& data){
    
    // approximation
    // lets init buffer with following size 
    // 61b + 3Nb | let N be 30 bytes each 
    //  = 151b -> 160b (round up)
    
    size_t total_size = 0;
    if (data.service.has_value())            total_size += 6;  // 1 + 4 + 1
    if (data.account_number.has_value())     total_size += 9;  // 1 + 4 + 4
    if (data.monetary_value.has_value())     total_size += 13; // 1 + 4 + 8
    if (data.currency.has_value())           total_size += 9;  // 1 + 4 + 4
    if (data.tx_account_number.has_value())  total_size += 9;  // 1 + 4 + 4
    
    if (data.account_owner_name.has_value()) 
        total_size += 5 + data.account_owner_name->size();
    if (data.account_password.has_value()) 
        total_size += 5 + data.account_password->size();
    if (data.tx_account_owner_name.has_value()) 
        total_size += 5 + data.tx_account_owner_name->size();

    return total_size;
}

std::optional<Protocol::FieldID> Protocol::CommandEncoder::to_field_id(uint8_t value) {
    switch (static_cast<Protocol::FieldID>(value)) {
        case FieldID::Service:
        case FieldID::AccountNumber:
        case FieldID::AccountOwnerName:
        case FieldID::AccountPassword:
        case FieldID::TxAccountNumber:
        case FieldID::TxAccountOwnerName:
        case FieldID::MonetaryValue:
        case FieldID::Currency:
        case FieldID::MonitorUpdates:
        case FieldID::MonitorTimeoutSeconds:
            return static_cast<FieldID>(value);
        default:
            return std::nullopt;
    }
}

bool Protocol::CommandEncoder::is_within_data_size(size_t offset,uint32_t length, const std::vector<uint8_t>& data){
    auto sum = Safe_math::safe_add(offset, length);
    if (!sum) return false;
    
    size_t s = *sum;
    if (s > data.size()) return false;
    
    return true;
}

void Protocol::CommandEncoder::append_uint8(std::vector<uint8_t> &buffer, uint8_t value){
    buffer.push_back(value);
}

void Protocol::CommandEncoder::append_uint16(std::vector<uint8_t> &buffer, uint16_t value){
    uint16_t networkValue = htons(value); 
    uint8_t* ptr = reinterpret_cast<uint8_t*>(&networkValue);
    buffer.insert(buffer.end(), ptr, ptr + sizeof(uint16_t));
}

void Protocol::CommandEncoder::append_uint32(std::vector<uint8_t> &buffer, uint32_t value){
    uint32_t networkValue = htonl(value); 
    uint8_t* ptr = reinterpret_cast<uint8_t*>(&networkValue);
    buffer.insert(buffer.end(), ptr, ptr + sizeof(uint32_t));
}

void Protocol::CommandEncoder::append_double(std::vector<uint8_t> &buffer, double value){
    uint64_t val;
    std::memcpy(&val, &value, sizeof(uint64_t));
    // Manual htonll
    // htonll is not consistently available across all Linux distributions 
    val = ((val & 0xFF00000000000000ULL) >> 56) |
        ((val & 0x00FF000000000000ULL) >> 40) |
        ((val & 0x0000FF0000000000ULL) >> 24) |
        ((val & 0x000000FF00000000ULL) >> 8)  |
        ((val & 0x00000000FF000000ULL) << 8)  |
        ((val & 0x0000000000FF0000ULL) << 24) |
        ((val & 0x000000000000FF00ULL) << 40) |
        ((val & 0x00000000000000FFULL) << 56);
    uint8_t* ptr = reinterpret_cast<uint8_t*>(&val);
    buffer.insert(buffer.end(), ptr, ptr + sizeof(uint64_t));
}

void Protocol::CommandEncoder::append_string(std::vector<uint8_t>& buffer, const std::string& str){
    buffer.insert(buffer.end(), str.begin(), str.end());
}

const std::unordered_map<Protocol::FieldID, Protocol::EncoderFunc> Protocol::CommandEncoder::encodeFuncMap = {
    {Protocol::FieldID::Service, &Protocol::CommandEncoder::encode_service},
    {Protocol::FieldID::AccountNumber, &Protocol::CommandEncoder::encode_account_number},
    {Protocol::FieldID::AccountOwnerName, &Protocol::CommandEncoder::encode_account_owner_name},
    {Protocol::FieldID::AccountPassword, &Protocol::CommandEncoder::encode_account_password},
    {Protocol::FieldID::TxAccountNumber, &Protocol::CommandEncoder::encode_tx_account_number},
    {Protocol::FieldID::TxAccountOwnerName, &Protocol::CommandEncoder::encode_tx_account_owner_name},
    {Protocol::FieldID::MonetaryValue, &Protocol::CommandEncoder::encode_monetary_value},
    {Protocol::FieldID::Currency, &Protocol::CommandEncoder::encode_currency},
    {Protocol::FieldID::MonitorUpdates, &Protocol::CommandEncoder::encode_monitor_updates},
    {Protocol::FieldID::MonitorTimeoutSeconds, &Protocol::CommandEncoder::encode_monitor_timeout_seconds}
};

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_service(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::Service);
    uint8_t value = static_cast<uint8_t>(data.service.value());
    uint32_t length = sizeof(uint8_t);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_uint8(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_account_number(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::AccountNumber);
    uint32_t value = data.account_number.value();
    uint32_t length = sizeof(uint32_t);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_uint32(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_account_owner_name(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::AccountOwnerName);
    std::string value = data.account_owner_name.value();
    uint32_t length = static_cast<uint32_t>(value.size());

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_string(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_account_password(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::AccountPassword);
    std::string value = data.account_password.value();
    uint32_t length = static_cast<uint32_t>(value.size());

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_string(buffer, value);

    return std::monostate{};
}   

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_tx_account_number(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::TxAccountNumber);
    uint32_t value = data.tx_account_number.value();
    uint32_t length = sizeof(uint32_t);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_uint32(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_tx_account_owner_name(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::TxAccountOwnerName);
    std::string value = data.tx_account_owner_name.value();
    uint32_t length = static_cast<uint32_t>(value.size());

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_string(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_monetary_value(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::MonetaryValue);
    double value = static_cast<double>(data.monetary_value.value());
    uint32_t length = sizeof(double);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_double(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_currency(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::Currency);
    uint8_t value = static_cast<uint8_t>(data.currency.value());
    uint32_t length = sizeof(uint8_t);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_uint8(buffer, value);
    
    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_monitor_updates(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::MonitorUpdates);
    std::string value = data.monitor_updates.value();
    uint32_t length = static_cast<uint32_t>(value.size());

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_string(buffer, value);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::encode_monitor_timeout_seconds(std::vector<uint8_t>& buffer, const Protocol::Command& data){
    uint8_t field_id = static_cast<uint8_t>(FieldID::MonitorTimeoutSeconds);
    uint32_t value = data.monitor_timeout_seconds.value();
    uint32_t length = sizeof(uint32_t);

    CommandEncoder::append_uint8(buffer, field_id);
    CommandEncoder::append_uint32(buffer, length);
    CommandEncoder::append_uint32(buffer, value);

    return std::monostate{};
}

const std::unordered_map<Protocol::FieldID, Protocol::DecoderFunc> Protocol::CommandEncoder::decodeFuncMap = {
    {Protocol::FieldID::Service, &Protocol::CommandEncoder::decode_service},
    {Protocol::FieldID::AccountNumber, &Protocol::CommandEncoder::decode_account_number},
    {Protocol::FieldID::AccountOwnerName, &Protocol::CommandEncoder::decode_account_owner_name},
    {Protocol::FieldID::AccountPassword, &Protocol::CommandEncoder::decode_account_password},
    {Protocol::FieldID::TxAccountNumber, &Protocol::CommandEncoder::decode_tx_account_number},
    {Protocol::FieldID::TxAccountOwnerName, &Protocol::CommandEncoder::decode_tx_account_owner_name},
    {Protocol::FieldID::MonetaryValue, &Protocol::CommandEncoder::decode_monetary_value},
    {Protocol::FieldID::Currency, &Protocol::CommandEncoder::decode_currency},
    {Protocol::FieldID::MonitorUpdates, &Protocol::CommandEncoder::decode_monitor_updates},
    {Protocol::FieldID::MonitorTimeoutSeconds, &Protocol::CommandEncoder::decode_monitor_timeout_seconds}
};

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_service(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){  
    if (length != sizeof(uint8_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    } // prevent buffer overflow (same for the following)

    uint8_t svc{};
    std::memcpy(&svc, buffer.data() + offset, length);
    data.service = static_cast<Protocol::Service>(svc);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length != sizeof(uint32_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    }
    
    uint32_t acc_num{};
    std::memcpy(&acc_num, buffer.data() + offset, length);
    acc_num = ntohl(acc_num);
    data.account_number = acc_num;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length > MAX_STRING_LENGTH) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_STRING_TOO_LONG);
    }
    
    std::string acc_own_name{};
    acc_own_name.resize(length);
    std::memcpy(acc_own_name.data(), buffer.data() + offset, length);
    data.account_owner_name = acc_own_name;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_account_password(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length > MAX_STRING_LENGTH) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_STRING_TOO_LONG);
    }

    std::string acc_pwd{};
    acc_pwd.resize(length);
    std::memcpy(acc_pwd.data(), buffer.data() + offset, length);
    data.account_password = acc_pwd;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_tx_account_number(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length != sizeof(uint32_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    }
    
    uint32_t tx_acc_num{};
    std::memcpy(&tx_acc_num, buffer.data() + offset, length);
    tx_acc_num = ntohl(tx_acc_num);
    data.tx_account_number = tx_acc_num;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_tx_account_owner_name(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length > MAX_STRING_LENGTH) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_STRING_TOO_LONG);
    }
    
    std::string tx_acc_name{};
    tx_acc_name.resize(length);
    std::memcpy(tx_acc_name.data(), buffer.data() + offset, length);
    data.tx_account_owner_name = tx_acc_name;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_monetary_value(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length != sizeof(uint64_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    }
    
    uint64_t val;
    std::memcpy(&val, buffer.data() + offset, length);

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
    data.monetary_value = mon_val;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_currency(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length != sizeof(uint8_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    }
    
    uint8_t cur{};
    std::memcpy(&cur, buffer.data() + offset, length);
    data.currency = static_cast<Protocol::CurrencyType>(cur);

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_monitor_updates(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length > MAX_STRING_LENGTH) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_STRING_TOO_LONG);
    }
    
    std::string monitor_updates{};
    monitor_updates.resize(length);
    std::memcpy(monitor_updates.data(), buffer.data() + offset, length);
    data.monitor_updates = monitor_updates;

    return std::monostate{};
}

Result<std::monostate, Error::InternalError> Protocol::CommandEncoder::decode_monitor_timeout_seconds(Command& data, size_t& offset, uint32_t length, const std::vector<uint8_t>& buffer){
    if (length != sizeof(uint32_t)) {
        return Result<std::monostate, Error::InternalError>::fail(
            Error::InternalError::DECODE_FIELD_OVERFLOW);
    }
    
    uint32_t monitor_timeout{};
    std::memcpy(&monitor_timeout, buffer.data() + offset, length);
    monitor_timeout = ntohl(monitor_timeout);
    data.monitor_timeout_seconds = monitor_timeout;

    return std::monostate{};
}