#include "internalError.h"

std::string Error::to_string(Error::InternalError err) {
    switch (err) {
        // Input / Application
        case Error::InternalError::USER_QUIT:                          return "USER_QUIT";
        case Error::InternalError::USER_CANCELED:                      return "USER_CANCELED";
        case Error::InternalError::BAD_INPUT:                          return "BAD_INPUT";
        case Error::InternalError::INVALID_SERVICE:                    return "INVALID_SERVICE";
        case Error::InternalError::INVALID_CURRENCY:                   return "INVALID_CURRENCY";

        // Network / Socket
        case Error::InternalError::INVALID_ADDRESS:                    return "INVALID_ADDRESS";
        case Error::InternalError::SOCKET_CREATE_FAILED:               return "SOCKET_CREATE_FAILED";
        case Error::InternalError::SEND_FAILED:                        return "SEND_FAILED";
        case Error::InternalError::RECEIVE_FAILED:                     return "RECEIVE_FAILED";

        // Request / Response
        case Error::InternalError::REQUEST_TIMEOUT:                    return "REQUEST_TIMEOUT";

        // Protocol Encoding
        case Error::InternalError::ENCODE_UNKNOWN_FIELD:               return "ENCODE_UNKNOWN_FIELD";
        case Error::InternalError::ENCODING_ERROR:                     return "ENCODING_ERROR";
        case Error::InternalError::DECODING_ERROR:                     return "DECODING_ERROR";
        case Error::InternalError::ENCODE_EMPTY_COMMAND:               return "ENCODE_EMPTY_COMMAND";
        case Error::InternalError::DECODE_EMPTY_DATA:                  return "DECODE_EMPTY_DATA";
        case Error::InternalError::DECODE_UNKNOWN_FIELD:               return "DECODE_UNKNOWN_FIELD";
        case Error::InternalError::DECODE_FIELD_OVERFLOW:              return "DECODE_FIELD_OVERFLOW";
        case Error::InternalError::DECODE_OFFSET_OVERFLOW:             return "DECODE_OFFSET_OVERFLOW";
        case Error::InternalError::DECODE_LENGTH_MISMATCH:             return "DECODE_LENGTH_MISMATCH";
        case Error::InternalError::DECODE_STRING_TOO_LONG:             return "DECODE_STRING_TOO_LONG";
        case Error::InternalError::DECODE_FIELD_MISMATCH:              return "DECODE_FIELD_MISMATCH";

        // Message Serialization
        case Error::InternalError::SERIALIZE_ERROR:                    return "SERIALIZE_ERROR";
        case Error::InternalError::DESERIALIZE_ERROR:                  return "DESERIALIZE_ERROR";
        case Error::InternalError::DESERIALIZE_HEADER_TOO_SHORT:       return "DESERIALIZE_HEADER_TOO_SHORT";
        case Error::InternalError::DESERIALIZE_PAYLOAD_OVERFLOW:       return "DESERIALIZE_PAYLOAD_OVERFLOW";
        case Error::InternalError::DESERIALIZE_PAYLOAD_INT_OVERFLOW:   return "DESERIALIZE_PAYLOAD_INT_OVERFLOW";

        default:                                                        return "UNKNOWN_ERROR";
    }
}