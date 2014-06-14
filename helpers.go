package admin

import (
	"crypto/rand"
	"reflect"
	"strconv"
	"strings"
)

func parseInt(s string) (int, error) {
	i64, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(i64), nil
}

// parseTag parses admin tags used in model structs.
// TODO: Report errors
func parseTag(s string) (map[string]string, error) {
	res := map[string]string{}

	inQuotes := false
	inKey := true

	var key string

	start := 0 // Where next key / value starts
	end := 0
	for i, c := range s {
		// Skip ahead if needed
		if i < start {
			continue
		}

		if inKey && c == '=' {
			// Key is complete, store it and look for value
			inKey = !inKey
			key = s[start:i]
			start = i + 1
		} else if c == '\'' && s[i-1] != '\'' {
			// For multi word values
			inQuotes = !inQuotes
			if inQuotes {
				start += 1
			}
			end = i
		}
		if (c == ' ' || i == len(s)-1) && !inQuotes {
			// Insert key and value. If only a key was found, insert as key with empty value.
			key = strings.TrimSpace(key)

			// If value is in quotes, end it one character earlier
			if end == 0 {
				end = i + 1
			}

			val := strings.TrimSpace(s[start:end])
			if len(key) == 0 {
				res[strings.TrimSpace(val)] = ""
			} else {
				res[key] = val
			}

			// Reset before starting to look for next pair
			start = i + 1
			end = 0
			key = ""
			inKey = true
		}
	}

	return res, nil
}

func randString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func typeToName(t reflect.Type) string {
	parts := strings.Split(t.String(), ".")
	return parts[len(parts)-1]
}

func typeToTableName(t reflect.Type, nameTransform NameTransformFunc) string {
	name := typeToName(t)

	if nameTransform != nil {
		return nameTransform(name)
	} else {
		return name
	}
}
