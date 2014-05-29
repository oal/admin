package admin

import (
	"testing"
)

func TestParseTagSimple(T *testing.T) {
	res, _ := parseTag("key=value")
	if key, ok := res["key"]; ok {
		if key != "value" {
			T.Error("Expected key to be 'value'")
		}
	} else {
		T.Error("Expected 'key' to be found.")
	}
}

func TestParseTagMulti(T *testing.T) {
	res, _ := parseTag("key=value another=pair and='another one'")
	if key, ok := res["another"]; ok {
		if key != "pair" {
			T.Error("Expected another to be 'pair'")
		}
	} else {
		T.Error("Expected 'another' to be found.")
	}
}
