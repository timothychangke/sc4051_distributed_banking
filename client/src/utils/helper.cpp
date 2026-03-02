#include "helper.h"

int safe_math::msb_index(unsigned int x) {
    if (x == 0) return -1;

    int index = 0;
    while (x >>= 1)
        ++index;

    return index;
}