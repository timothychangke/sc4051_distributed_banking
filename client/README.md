## Client

### Instructions to compile the code

cmake is used to compile the code across different OSes.
Follow the instructions below for the environment you are using.

#### WSL Ubuntu / Linux

```bash
mkdir -p build
cd build

cmake ..
cmake --build .

./client 127.0.0.1 2222 -m
```

Note: on WSL Ubuntu, CMake usually generates a single executable in the build directory.
It does not normally create a Debug folder unless you explicitly use a multi-config generator.
