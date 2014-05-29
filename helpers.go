package admin

import (
	"fmt"
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

// Parse admin tags used in model structs.
func parseTag(s string) (map[string]string, error) {
	res := map[string]string{}

	inQuotes := false
	inKey := true

	var key string

	start := 0 // Where next key / value starts
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
			fmt.Println(key)
		} else if c == '\'' && s[i-1] != '\'' {
			// For multi word values
			inQuotes = !inQuotes
			if inQuotes {
				start += 1
			}
			fmt.Println("QUotes", inQuotes)
		}
		if (c == ' ' || i == len(s)-1) && !inQuotes {
			// Insert key and value. If only a key was found, insert as key with empty value.
			key = strings.TrimSpace(key)
			val := s[start:i]
			fmt.Println(val, len(val))
			if len(key) == 0 {
				res[strings.TrimSpace(val)] = ""
			} else {
				res[key] = val
			}
			start = i + 1
			key = ""
			inKey = true
		}
	}

	return res, nil
}
