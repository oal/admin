package admin

import (
	"fmt"
	"reflect"
	"strings"
)

// queryModel is used in list view to display all rows.
func (a *Admin) queryModel(mdl *model) ([][]interface{}, error) {
	q := fmt.Sprintf("SELECT id, %v FROM %v", strings.Join(mdl.listTableColumns(), ","), mdl.tableName)
	rows, err := a.db.Query(q)
	if err != nil {
		return nil, err
	}

	numCols := len(mdl.listTableColumns()) + 1
	results := [][]interface{}{}

	for rows.Next() {
		result, err := scanRow(numCols, rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// querySingleModel is used in edit view.
func (a *Admin) querySingleModel(mdl *model, id int) ([]interface{}, error) {
	numCols := len(mdl.fieldNames()) + 1
	q := fmt.Sprintf("SELECT id, %v FROM %v WHERE id = ?", strings.Join(mdl.tableColumns(), ","), mdl.tableName)
	row := a.db.QueryRow(q, id)

	result, err := scanRow(numCols, row)
	if err != nil {
		return nil, err
	}

	return result, nil
}

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
