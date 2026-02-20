extends GdUnitTestSuite

# TODO: implement a file with intentional syntax errors to trigger a compile error
# Example (intentional syntax error):
# func test_broken() -> void:
#     var x = (  # unclosed parenthesis - causes parse error


func test_broken() -> void:
	var x = (  # unclosed parenthesis - causes parse error
