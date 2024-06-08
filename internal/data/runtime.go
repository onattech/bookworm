package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Define an error that our UnmarshalJSON() method can return if we're unable to parse
// or convert the JSON string successfully.
var ErrInvalidISBNFormat = errors.New("invalid ISBN format")

// Declare a custom ISBN type, which has the underlying type int32 (the same as our
// Book struct field).
type ISBN int32

// Implement a MarshalJSON() method on the ISBN type so that it satisfies the
// json.Marshaler interface. This should return the JSON-encoded value for the book
// ISBN (in our case, it will return a string in the format "1-4028-9462-7" or "978-0-306-40615-7").
func (r ISBN) MarshalJSON() ([]byte, error) {
	// Generate a string containing the book ISBN in the required format.
	jsonValue := fmt.Sprintf("%d -", r)

	// Use the strconv.Quote() function on the string to wrap it in double quotes. It
	// needs to be surrounded by double quotes in order to be a valid *JSON string*.
	quotedJSONValue := strconv.Quote(jsonValue)

	// Convert the quoted string value to a byte slice and return it.
	return []byte(quotedJSONValue), nil
}

// Implement a UnmarshalJSON() method on the ISBN type so that it satisfies the
// json.Unmarshaler interface. IMPORTANT: Because UnmarshalJSON() needs to modify the
// receiver (our ISBN type), we must use a pointer receiver for this to work
// correctly. Otherwise, we will only be modifying a copy (which is then discarded when
// this method returns).
func (r *ISBN) UnmarshalJSON(jsonValue []byte) error {
	// We expect that the incoming JSON value will be a string in the format
	// "978-0-306-40615-7", and the first thing we need to do is remove the "-"s
	// hyphens from this string. If we can't unquote it, then we return the
	// ErrInvalidISBNFormat error.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidISBNFormat
	}

	// Remove hyphens
	cleanedISBN := strings.ReplaceAll(unquotedJSONValue, "-", "")

	// Otherwise, parse the string containing the number into an int32. Again, if this
	// fails return the ErrInvalidISBNFormat error.
	i, err := strconv.ParseInt(cleanedISBN, 10, 32)
	if err != nil {
		return ErrInvalidISBNFormat
	}

	// Convert the int32 to a ISBN type and assign this to the receiver. Note that we
	// use the * operator to deference the receiver (which is a pointer to a ISBN
	// type) in order to set the underlying value of the pointer.
	*r = ISBN(i)

	return nil
}
