package admin

import (
	"fmt"
	"strings"
)

// queryModel is used in list view to display all rows.
func (a *Admin) queryModel(mdl *model) ([][]string, error) {
	q := fmt.Sprintf("SELECT id, %v FROM %v", strings.Join(mdl.listTableColumns(), ","), mdl.tableName)
	rows, err := a.db.Query(q)
	if err != nil {
		return nil, err
	}

	numCols := len(mdl.listTableColumns()) + 1
	results := [][]string{}

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
func (a *Admin) querySingleModel(mdl *model, id int) ([]string, error) {
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
func scanRow(numCols int, scanner MultiScanner) ([]string, error) {
	rawResult := make([][]byte, numCols)
	dest := make([]interface{}, numCols)
	for i, _ := range rawResult {
		dest[i] = &rawResult[i]
	}

	err := scanner.Scan(dest...)
	if err != nil {
		return nil, err
	}

	result := make([]string, numCols)

	for i, raw := range rawResult {
		if raw == nil {
			result[i] = "\\N"
		} else {
			result[i] = string(raw)
		}
	}

	return result, nil
}
