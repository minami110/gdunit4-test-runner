extends GdUnitTestSuite

func test_addition() -> void:
	assert_int(1 + 1).is_equal(2)
	
func test_subtraction() -> void:
	assert_int(5 - 3).is_equal(2)
