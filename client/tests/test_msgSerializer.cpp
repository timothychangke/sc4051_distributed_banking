#include <gtest/gtest.h>
#include <gmock/gmock.h>

#include "msgSerializer.h"
#include "result.h"
#include "internalError.h"


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
    EXPECT_TRUE(serializer->validate_header(18)); 
    EXPECT_TRUE(serializer->validate_header(20)); 
}

TEST_F(MessageSerializerTest, validate_header_too_short){
    EXPECT_FALSE(serializer->validate_header(17));
    EXPECT_FALSE(serializer->validate_header(10));
}

TEST_F(MessageSerializerTest, validate_payload_valid){
    EXPECT_TRUE(serializer->validate_payload(20u, 10u, 10u));
}

TEST_F(MessageSerializerTest, validate_payload_overflow){
    EXPECT_FALSE(serializer->validate_payload(20u, 10u, 20u));
}

TEST_F(MessageSerializerTest, validate_payload_safe_add_overflow){
    uint32_t max_u32 = (std::numeric_limits<uint32_t>::max)();
    EXPECT_FALSE(serializer->validate_payload(100u, 100u, max_u32));
}

TEST_F(MessageSerializerTest, serialize_success) {
    Protocol::Message msg;
    msg.type = Protocol::MessageType::Request;
    msg.flag = Semantics::InvocationFlag::AT_LEAST_ONCE;
    msg.id.request_id = 12345;
    msg.id.port = 8080;
    msg.payload.status_code = 200;
    msg.payload.content = {'H', 'e', 'l', 'l', 'o'};

    auto res = serializer->serialize(msg);
    ASSERT_TRUE(res.ok());
    
    std::vector<uint8_t> data = res.value();
    EXPECT_EQ(data.size(), 23);
    EXPECT_EQ(data[0], static_cast<uint8_t>(Protocol::MessageType::Request));
    EXPECT_EQ(data[1], static_cast<uint8_t>(Semantics::InvocationFlag::AT_LEAST_ONCE));
}

TEST_F(MessageSerializerTest, deserialize_success) {
    std::vector<uint8_t> data = {
    };

    auto res = serializer->deserialize(data);
    ASSERT_TRUE(res.ok());
    
    Protocol::Message msg = res.value();
    EXPECT_EQ(msg.type, Protocol::MessageType::Request);
    EXPECT_EQ(msg.flag, Semantics::InvocationFlag::AT_MOST_ONCE);
    EXPECT_EQ(msg.id.request_id, 12345);
    EXPECT_EQ(msg.id.port, 8080);
    EXPECT_EQ(msg.payload.status_code, 200);
    std::string content(msg.payload.content.begin(), msg.payload.content.end());
    EXPECT_EQ(content, "Hello");
}

TEST_F(MessageSerializerTest, deserialize_header_too_short) {
    std::vector<uint8_t> data(17, 0);

    auto res = serializer->deserialize(data);
    ASSERT_FALSE(res.ok());
    EXPECT_EQ(res.error(), Error::InternalError::DESERIALIZE_HEADER_TOO_SHORT);
}

TEST_F(MessageSerializerTest, deserialize_payload_overflow) {
    std::vector<uint8_t> data = {
        static_cast<uint8_t>(Protocol::MessageType::Request),
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

TEST_F(MessageSerializerTest, round_trip_consistency) {
    Protocol::Message original;
    original.type = Protocol::MessageType::Reply;
    original.flag = Semantics::InvocationFlag::AT_MOST_ONCE;
    original.id.request_id = 987654;
    original.id.ipv4_address = 0xC0A80101; // 192.168.1.1
    original.id.port = 54321;
    original.payload.status_code = 404;
    original.payload.content = {'N', 'o', 't', ' ', 'F', 'o', 'u', 'n', 'd'};

    // Serialize
    auto ser_res = serializer->serialize(original);
    ASSERT_TRUE(ser_res.ok());
    
    // Deserialize
    auto deser_res = serializer->deserialize(ser_res.value());
    ASSERT_TRUE(deser_res.ok());
    
    Protocol::Message result = deser_res.value();
    
    // Compare
    EXPECT_EQ(result.type, original.type);
    EXPECT_EQ(result.flag, original.flag);
    EXPECT_EQ(result.id.request_id, original.id.request_id);
    EXPECT_EQ(result.id.ipv4_address, original.id.ipv4_address);
    EXPECT_EQ(result.id.port, original.id.port);
    EXPECT_EQ(result.payload.status_code, original.payload.status_code);
    EXPECT_EQ(result.payload.content, original.payload.content);
}
