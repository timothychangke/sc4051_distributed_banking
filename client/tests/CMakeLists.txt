include(FetchContent)
FetchContent_Declare(
  googletest
  URL https://github.com/google/googletest/archive/03597a01ee50ed33e9dfd640b249b4be3799d395.zip
)
# For Windows: Prevent overriding the parent project's compiler/linker settings
set(gtest_force_shared_crt ON CACHE BOOL "" FORCE)
FetchContent_MakeAvailable(googletest)

# Register GoogleTest
include(GoogleTest)

# BankClient Tests
add_executable(test_bankClient test_bankClient.cpp)
target_link_libraries(test_bankClient gtest_main gmock_main client_lib)
gtest_discover_tests(test_bankClient)

# NetworkUtils Tests
add_executable(test_networkUtils test_networkUtils.cpp)
target_link_libraries(test_networkUtils gtest_main gmock_main client_lib)
gtest_discover_tests(test_networkUtils)
