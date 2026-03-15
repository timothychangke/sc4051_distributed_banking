#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "msgSerializer.h"
#include "result.h"
#include "internalError.h"

/*
    This test file test for the following functions: 
    -----------------------------------------------

    serialize(const Message& message);
    deserialize(const std::vector<uint8_t>& data);
    validate_header(size_t header_size); 
    validate_payload(size_t payload_size, size_t offset, uint32_t content_len); 

*/