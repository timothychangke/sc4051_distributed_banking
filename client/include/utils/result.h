#pragma once

#include <variant>

/**
 * Result<T, E>
 *
 * Represents either a success value (T) or an error (E).
 * Modelled after std::expected (C++23) but compatible with C++17.
 *
 * Usage:
 *   Result<Message, Error::InternalError> deserialize(...);
 *
 *   auto res = deserialize(data);
 *   if (!res) return Result<Foo, Error::InternalError>::fail(res.error());
 *   auto& msg = res.value();
 */
template <typename T, typename E>
class Result {
    std::variant<T, E> _data;
public:
    Result() = default;

    // Construct a success result (implicit, allows: return myValue;)
    Result(T val) : _data(std::move(val)) {}

    // Construct an error result
    static Result fail(E err) {
        Result r;
        r._data = std::move(err);
        return r;
    }

    // True if holding a success value
    bool ok() const { return std::holds_alternative<T>(_data); }
    explicit operator bool() const { return ok(); }

    // Access the success value (only call if ok())
    T&       value()       { return std::get<T>(_data); }
    const T& value() const { return std::get<T>(_data); }

    // Access the error (only call if !ok())
    E        error() const { return std::get<E>(_data); }

    // Equality operators for testing
    bool operator==(const Result& other) const {
        return _data == other._data;
    }
    bool operator==(const T& val) const {
        return ok() && value() == val;
    }
};
