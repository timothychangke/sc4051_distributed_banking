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

TEST_F(MessageSerializerTest, validate_header_valid){
    // HEADER_SIZE IS 17 BYTES
    EXPECT_TRUE(serializer->validate_header(20)); 
}

TEST_F(MessageSerializerTest, validate_header_too_short){
    // HEADER_SIZE IS 17 BYTES
    EXPECT_FALSE(serializer->validate_header(10));
}

TEST_F(MessageSerializerTest, validate_payload_valid){
    EXPECT_TRUE(serializer->validate_payload(20u, 10u, 10u));
}

TEST_F(MessageSerializerTest, validate_payload_overflow){
    // sum (offset + content_len) is larger than payload's total_size
    EXPECT_FALSE(serializer->validate_payload(20u, 10u, 20u));
}

TEST_F(MessageSerializerTest, serialize_success) {
    Protocol::Message msg;
    msg.type = Protocol::MessageType::Request;
    msg.id.request_id = 12345;
    msg.id.ipv4_address = 0x7F000001; // 127.0.0.1
    msg.id.port = 8080;
    msg.payload.status_code = 200;
    msg.payload.content = {'H', 'e', 'l', 'l', 'o'};

    auto res = serializer->serialize(msg);
    ASSERT_TRUE(res.ok());
    
    std::vector<uint8_t> data = res.value();
    // 1 (type) + 4 (id) + 4 (ip) + 2 (port) + 2 (sc) + 4 (len) + 5 (content) = 22 bytes
    EXPECT_EQ(data.size(), 22);
    EXPECT_EQ(data[0], static_cast<uint8_t>(Protocol::MessageType::Request));
}

TEST_F(MessageSerializerTest, deserialize_success) {
    // 22 bytes matching the structure above
    std::vector<uint8_t> data = {
        static_cast<uint8_t>(Protocol::MessageType::Request), // Type (1)
        0, 0, 0x30, 0x39, // ID: 12345
        0x7F, 0, 0, 0x01, // IP: 127.0.0.1 (4)
        0x1F, 0x90,       // Port: 8080 (2)
        0x00, 0xC8,       // Status: 200 (2)
        0, 0, 0, 5,       // Len: 5 (4)
        'H', 'e', 'l', 'l', 'o' // Content (5)
    };

    auto res = serializer->deserialize(data);
    ASSERT_TRUE(res.ok());
    
    Protocol::Message msg = res.value();
    EXPECT_EQ(msg.type, Protocol::MessageType::Request);
    EXPECT_EQ(msg.id.request_id, 12345);
    EXPECT_EQ(msg.id.port, 8080);
    EXPECT_EQ(msg.payload.status_code, 200);
    std::string content(msg.payload.content.begin(), msg.payload.content.end());
    EXPECT_EQ(content, "Hello");
}

TEST_F(MessageSerializerTest, deserialize_header_too_short) {
    // Header is 17 bytes, so 16 bytes is too short
    std::vector<uint8_t> data(16, 0);

    auto res = serializer->deserialize(data);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DESERIALIZE_HEADER_TOO_SHORT);
}

TEST_F(MessageSerializerTest, deserialize_payload_overflow) {
    // Valid header but content_len claims 10 bytes while only 5 are provided
    std::vector<uint8_t> data = {
        static_cast<uint8_t>(Protocol::MessageType::Request),
        0, 0, 0x30, 0x39,       // ID
        0x7F, 0, 0, 0x01,       // IP
        0x1F, 0x90,             // Port
        0x00, 0xC8,             // Status
        0, 0, 0, 10,            // Content Length CLAIMED: 10
        'H', 'e', 'l', 'l', 'o' // Content PROVIDED: 5 bytes
    };

    auto res = serializer->deserialize(data);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DESERIALIZE_PAYLOAD_OVERFLOW);
}