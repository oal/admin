package admin

import (
	"reflect"
)

// MultiScanner is like the db.Scan interface, but scans to a slice.
type MultiScanner interface {
	Scan(src ...interface{}) error
}

// scanRow loads all data from a row into a string slice.
func scanRow(numCols int, scanner MultiScanner) ([]interface{}, error) {
	// We can only scan into pointers, so create result and destination slices
	result := make([]interface{}, numCols)
	dest := make([]interface{}, numCols)
	for i, _ := range result {
		dest[i] = &result[i]
	}

	err := scanner.Scan(dest...)
	if err != nil {
		return nil, err
	}

	// These are *interface{}, so get the interface{} and check if we can convert byte slice to string
	for i := 0; i < numCols; i++ {
		val := reflect.ValueOf(dest[i]).Elem().Interface()
		if str, ok := val.([]uint8); ok {
			result[i] = string(str)
		}
	}

	return result, nil
}
