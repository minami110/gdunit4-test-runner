extends GdUnitTestSuite


func test_string_concat() -> void:
	assert_str("hello" + " " + "world").is_equal("hello world")

func test_string_length() -> void:
	assert_int("gdunit4".length()).is_equal(7)
