#pragma once

#include <string>

namespace Error{
enum class InternalError {
    // Input / Application Layer  
    USER_QUIT,                          // User quit/exit the process
    USER_CANCELED,                      // User canceled the process
    BAD_INPUT,                          // User provided invalid or unparseable input
    BAD_PW_LEN,                         // User provided password exceeding MAX_PW_LEN
    INVALID_SERVICE,                    // Service type selected is not recognised
    INVALID_CURRENCY,                   // Currency string does not map to a known CurrencyType

    // Network / Socket Layer
    INVALID_ADDRESS,                    // IPv4 address string could not be parsed by inet_pton
    SOCKET_CREATE_FAILED,               // socket() syscall returned -1 (fd creation failed)
    SEND_FAILED,                        // sendto() returned a negative value
    RECEIVE_FAILED,                     // recvfrom() returned a negative value

    // Request / Response
    REQUEST_TIMEOUT,                    // No response received within the retry window

    // Protocol Encoding Layer  
    ENCODING_ERROR,                     // General failure while encoding a Command to bytes
    DECODING_ERROR,                     // General failure while decoding bytes to a Command

    ENCODE_UNKNOWN_FIELD,               // Encountered a field_id that does not map to a known FieldID
    ENCODE_EMPTY_COMMAND,               // Attempted to encode a Command with no fields set
    DECODE_EMPTY_DATA,                  // Received an empty byte buffer for decoding

    DECODE_UNKNOWN_FIELD,               // Encountered a byte that does not map to a known FieldID
    DECODE_FIELD_OVERFLOW,              // Field length + offset would exceed the data buffer size
    DECODE_OFFSET_OVERFLOW,             // Integer overflow when advancing the decode offset
    DECODE_LENGTH_MISMATCH,             // Fixed-size field declared a length that does not match its type size
    DECODE_STRING_TOO_LONG,             // String field length exceeds MAX_STRING_LENGTH
    DECODE_FIELD_MISMATCH,              // Declared field length does not match the expected size for this field type

    // Message Serialization Layer
    SERIALIZE_ERROR,                    // General failure while serializing a Message to bytes
    DESERIALIZE_ERROR,                  // General failure while deserializing bytes to a Message
    DESERIALIZE_HEADER_TOO_SHORT,       // Received data is smaller than HEADER_SIZE
    DESERIALIZE_PAYLOAD_OVERFLOW,       // content_len + offset would exceed total data size
    DESERIALIZE_PAYLOAD_INT_OVERFLOW,   // Integer overflow computing payload end offset

};

std::string to_string(InternalError err);
}