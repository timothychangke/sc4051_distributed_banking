cmake_minimum_required(VERSION 3.15)
project(client LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED True)

# Recursive glob for all sources
file(GLOB_RECURSE ALL_SOURCES CONFIGURE_DEPENDS "src/*.cpp")

# Exclude main.cpp from the library sources
set(LIB_SOURCES ${ALL_SOURCES})
list(REMOVE_ITEM LIB_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/src/main.cpp")

# Link against Winsock on Windows
if(WIN32)
    set(SOCKET_LIBRARIES Ws2_32)
endif()

# Create a static library with all the client logic (so tests can link to it)
add_library(client_lib STATIC ${LIB_SOURCES})

# Comprehensive include paths
target_include_directories(client_lib PUBLIC 
    include
    include/common
    include/core
    include/networkUtils
    include/protocol
    include/utils
)

target_link_libraries(client_lib PUBLIC ${SOCKET_LIBRARIES})

# Create the main executable and link the library
add_executable(client src/main.cpp)
target_link_libraries(client PRIVATE client_lib)

# Testing setup
enable_testing()
add_subdirectory(tests)