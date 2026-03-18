#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "udpSocket.h"

#ifdef _WIN32
#include <winsock2.h>
#endif

/**
 * NetworkIntegrationTest
 * 
 * Verifies actual UDP communication using the loopback interface (127.0.0.1).
 */
class NetworkIntegrationTest : public ::testing::Test {
protected:
    void SetUp() override {
#ifdef _WIN32
        WSADATA wsaData;
        int res = WSAStartup(MAKEWORD(2, 2), &wsaData);
        if (res != 0) {
            FAIL() << "WSAStartup failed with error: " << res;
        }
#endif
    }

    void TearDown() override {
#ifdef _WIN32
        WSACleanup();
#endif
    }
};

TEST_F(NetworkIntegrationTest, SendAndReceiveLoopback) {
    uint16_t test_port = 9999;
    
    // Create Receiver - it will listen on test_port
    NetworkUtils::UDPSocket receiver("127.0.0.1", test_port, false);
    auto bind_res = receiver.bind_socket();
    ASSERT_TRUE(bind_res.ok()) << "Failed to bind receiver socket to port " << test_port;

    // Create Sender - it will send to test_port
    NetworkUtils::UDPSocket sender("127.0.0.1", test_port);

    std::string msg = "Hello World";
    std::vector<uint8_t> send_data(msg.begin(), msg.end());

    auto send_res = sender.send_message(send_data);
    ASSERT_TRUE(send_res.ok()) << "Failed to send UDP message";

    auto recv_res = receiver.receive_message();
    ASSERT_TRUE(recv_res.ok()) << "Failed to receive UDP message";

    EXPECT_EQ(recv_res.value(), send_data);
    std::string received_msg(recv_res.value().begin(), recv_res.value().end());
    EXPECT_EQ(received_msg, msg);
}

TEST_F(NetworkIntegrationTest, ReceiveTimeoutOrFailure) {
    
    uint16_t test_port = 9998;
    NetworkUtils::UDPSocket receiver("127.0.0.1", test_port, false);
    ASSERT_TRUE(receiver.bind_socket().ok());

    NetworkUtils::UDPSocket sender("127.0.0.1", test_port);

    // Send an empty packet
    std::vector<uint8_t> empty_data;
    ASSERT_TRUE(sender.send_message(empty_data).ok());

    auto recv_res = receiver.receive_message();
    ASSERT_TRUE(recv_res.ok());
    EXPECT_TRUE(recv_res.value().empty());
}
