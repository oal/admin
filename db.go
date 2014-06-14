package admin

import (
	"database/sql"
	"fmt"
	"github.com/oal/admin/fields"
	"reflect"
	"strings"
)

// queryModel is used in list view to display all rows.
func (a *Admin) queryModel(mdl *model, search string) ([][]interface{}, error) {
	// Ugly search. Will fix later.
	qSearch := ""
	doSearch := false
	var searchList []interface{}
	if len(search) > 0 {
		if len(mdl.searchableColumns) > 0 {
			searchCols := make([]string, len(mdl.searchableColumns))
			searchList = make([]interface{}, len(searchCols))
			for i, _ := range searchList {
				searchList[i] = search
			}
			for i, _ := range searchCols {
				searchCols[i] = fmt.Sprintf("%v LIKE ?", mdl.searchableColumns[i])
			}
			qSearch = fmt.Sprintf("WHERE %v", strings.Join(searchCols, " OR "))
			doSearch = true
		}

	}

	cols := []string{}
	tables := []string{mdl.tableName}
	where := []string{}
	for _, field := range mdl.fields {
		if field.Attrs().List {
			colName := fmt.Sprintf("%v.%v", mdl.tableName, field.Attrs().ColumnName)
			if fk, ok := field.(*fields.ForeignKeyField); ok && len(fk.ListColumn) > 0 {
				fkColName := fmt.Sprintf("%v.%v", fk.TableName, fk.ListColumn)
				where = append(where, fmt.Sprintf("%v = %v.id", colName, fk.TableName))
				colName = fkColName
				tables = append(tables, fk.TableName)
			}
			cols = append(cols, colName)
		}
	}
	whereStr := ""
	if len(where) > 0 {
		whereStr = fmt.Sprintf("WHERE %v", strings.Join(where, " AND "))
	}

	q := fmt.Sprintf("SELECT %v FROM %v %v %v", strings.Join(cols, ", "), strings.Join(tables, ", "), whereStr, qSearch)

	var rows *sql.Rows
	var err error
	if doSearch {
		rows, err = a.db.Query(q, searchList...)
	} else {
		rows, err = a.db.Query(q)
	}

	if err != nil {
		return nil, err
	}

	numCols := len(cols)
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
func (a *Admin) querySingleModel(mdl *model, id int) (map[string]interface{}, error) {
	numCols := len(mdl.fieldNames)

	// Can't do * as column order in the DB might not match struct
	cols := make([]string, numCols)
	for i, fieldName := range mdl.fieldNames {
		if a.NameTransform != nil {
			fieldName = a.NameTransform(fieldName)
		}
		cols[i] = fieldName
	}

	q := fmt.Sprintf("SELECT %v FROM %v WHERE id = ?", strings.Join(cols, ", "), mdl.tableName)
	row := a.db.QueryRow(q, id)

	result, err := scanRow(numCols, row)
	if err != nil {
		return nil, err
	}

	resultMap := map[string]interface{}{}
	for i, val := range result {
		resultMap[mdl.fieldNames[i]] = val
	}

	return resultMap, nil
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
