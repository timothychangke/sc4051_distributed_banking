#pragma once 

#include <optional>
#include <type_traits>
#include <limits>

namespace Safe_math  {
    int msb_index(unsigned int x);

    template <typename T, typename U>
    auto safe_add(T a, U b) -> std::optional<std::common_type_t<T, U>> {
        using R = std::common_type_t<T, U>;
        R ra = static_cast<R>(a);
        R rb = static_cast<R>(b);
        R result = ra + rb;

        if (std::is_signed<R>::value) {
            // signed overflow
            if (((ra ^ result) & (rb ^ result)) < 0) return std::nullopt;
        } else {
            // unsigned overflow (works for size_t)
            if (result < ra || result < rb) return std::nullopt;
        }

        return result;
    }

    template <typename T, typename U>
    auto safe_minus(T a, U b) -> std::optional<std::common_type_t<T, U>> {
        using R = std::common_type_t<T, U>;
        R ra = static_cast<R>(a);
        R rb = static_cast<R>(b);
        R result = ra - rb;

        if (std::is_signed<R>::value) {
            // signed overflow
            if (((ra ^ rb) & (ra ^ result)) < 0) return std::nullopt;
        } else {
            // unsigned underflow (works for size_t)
            if (rb > ra) return std::nullopt;
        }

        return result;
    }
}