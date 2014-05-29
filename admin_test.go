package admin

import (
	"testing"
)

func TestParseTagSimple(T *testing.T) {
	res, _ := parseTag("key=value only_key")
	if key, ok := res["key"]; ok {
		if key != "value" {
			T.Error("Expected 'key' to be 'value'")
		}
	} else {
		T.Error("Expected 'key' to be found.")
	}

	if key, ok := res["only_key"]; ok {
		if key != "" {
			T.Error("Expected 'only_key' to be empty")
		}
	} else {
		T.Error("Expected 'only_key' to be found.")
	}
}

func TestParseTagMulti(T *testing.T) {
	res, _ := parseTag("key=value another=pair and='another one'")
	if key, ok := res["another"]; ok {
		if key != "pair" {
			T.Error("Expected 'another' to be 'pair'")
		}
	} else {
		T.Error("Expected 'another' to be found.")
	}

	if key, ok := res["and"]; ok {
		if key != "another one" {
			T.Error("Expected 'and' to be 'another one'")
		}
	} else {
		T.Error("Expected 'and' to be found.")
	}
}
