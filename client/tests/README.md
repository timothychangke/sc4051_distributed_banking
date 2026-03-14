## GTEST 

Google C++ testing framework **gtest** is an industry standard for writing robust and maintainable code. 

### Google Test Macro Quick Reference

| Macro | Purpose |
| :--- | :--- |
| `EXPECT_EQ(val1, val2)` | Verify that $val1 == val2$. |
| `EXPECT_TRUE(condition)` | Verify that the given condition is **true**. |
| `EXPECT_STREQ(s1, s2)` | Compare two C-style strings (checks actual content, not just pointers). |
| `EXPECT_THROW(statement, exception)` | Verify that a specific code statement throws a designated exception type. |
| `TEST_F(ClassName, Test)` | Define a test that uses a **Test Fixture** (shared `SetUp` and `TearDown` logic). |

---

### Assertions vs. Expectations
Replace `EXPECT_` with `ASSERT_` for any of the above if you want the test to **abort immediately** upon failure.

* **`EXPECT_...`**: Logs a failure but continues the rest of the test.
* **`ASSERT_...`**: Logs a failure and exits the current test function immediately.