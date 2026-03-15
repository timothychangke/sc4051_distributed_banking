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

class MessageSerializerTestWrapper : public Protocol::MessageSerializer {
public :
    using MessageSerializer::validate_header;
    using MessageSerializer::validate_payload;
};

class MessageSerializerTest : public ::testing::Test {
protected:
    MessageSerializerTestWrapper* serializer;
    void SetUp() override {
        serializer = new MessageSerializerTestWrapper();
    }
    void TearDown() override {
        delete serializer;
    }
};

TEST_F(MessageSerializerTest, validate_header_true){
    // HEADER_SIZE IS 17 BYTES
    EXPECT_TRUE(serializer->validate_header(20)); 
}

TEST_F(MessageSerializerTest, validate_header_false){
    // HEADER_SIZE IS 17 BYTES
    EXPECT_FALSE(serializer->validate_header(10));
}

TEST_F(MessageSerializerTest, validate_payload_true){
    EXPECT_TRUE(serializer->validate_payload(20u, 10u, 10u));
}

TEST_F(MessageSerializerTest, validate_payload_false){
    // sum (offset + content_len) is larger than payload's total_size
    EXPECT_FALSE(serializer->validate_payload(20u, 10u, 20u));
}