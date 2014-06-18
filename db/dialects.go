package db

import (
	"bytes"
	"fmt"
	"strings"
)

type Dialect interface {
	Queryf(format string, args ...interface{}) string
}

type BaseDialect struct{}

func (BaseDialect) Queryf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

type PostgresDialect struct{}

func (PostgresDialect) Queryf(format string, args ...interface{}) string {
	parts := strings.Split(format, "?")
	var buf bytes.Buffer

	for i, part := range parts {
		buf.WriteString(part)
		if i < len(parts)-1 {
			buf.WriteString(fmt.Sprintf("$%d", i+1))
		}
	}
	return fmt.Sprintf(buf.String(), args...)
}
