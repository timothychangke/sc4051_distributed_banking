cmake_minimum_required(VERSION 3.15)
project(client LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED True)

# Recursive glob for all sources
file(GLOB_RECURSE SOURCES CONFIGURE_DEPENDS "src/*.cpp")

# Link against Winsock on Windows
if(WIN32)
    set(SOCKET_LIBRARIES Ws2_32)
endif()

add_executable(client ${SOURCES})

# Comprehensive include paths
target_include_directories(client PRIVATE 
    include
    include/common
    include/core
    include/networkUtils
    include/protocol
    include/utils
)

target_link_libraries(client PRIVATE ${SOCKET_LIBRARIES})