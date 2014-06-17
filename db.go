package admin

import (
	"database/sql"
	"fmt"
	"github.com/oal/admin/fields"
	"reflect"
	"strings"
)

// queryModel is used in list view to display all rows.
func (a *Admin) queryModel(mdl *model, search, sortBy string, sortDesc bool, page int) ([][]interface{}, int, error) {
	page--

	// Ugly search. Will fix later.
	doSearch := false
	whereStr := ""
	var searchList []interface{}
	if len(search) > 0 {
		if len(mdl.searchableColumns) > 0 {
			searchCols := make([]string, len(mdl.searchableColumns))
			searchList = make([]interface{}, len(searchCols))
			for i, _ := range searchList {
				searchList[i] = search
			}
			for i, _ := range searchCols {
				searchCols[i] = fmt.Sprintf("%v.%v LIKE ?", mdl.tableName, mdl.searchableColumns[i])
			}
			whereStr = fmt.Sprintf("WHERE (%v)", strings.Join(searchCols, " OR "))
			doSearch = true
		}

	}

	cols := []string{}
	tables := []string{mdl.tableName}
	fkWhere := []string{}
	for _, field := range mdl.fields {
		if field.Attrs().List {
			colName := fmt.Sprintf("%v.%v", mdl.tableName, field.Attrs().ColumnName)
			if relField, ok := field.(fields.RelationalField); ok && len(relField.GetListColumn()) > 0 {
				relTable := relField.GetRelatedTable()
				fkColName := fmt.Sprintf("%v.%v", relTable, relField.GetListColumn())
				fkWhere = append(fkWhere, fmt.Sprintf("%v = %v.id", colName, relTable))
				colName = fkColName
				tables = append(tables, relTable)
			}
			cols = append(cols, colName)
		}
	}

	if len(fkWhere) > 0 {
		if len(whereStr) > 0 {
			whereStr += fmt.Sprintf(" AND (%v)", strings.Join(fkWhere, " AND "))
		} else {
			whereStr = "WHERE " + strings.Join(fkWhere, " AND ")
		}
	}

	sqlColumns := strings.Join(cols, ", ")
	sqlTables := strings.Join(tables, ", ")

	if len(sortBy) > 0 {
		sortCol := sortBy
		if a.NameTransform != nil {
			sortCol = a.NameTransform(sortBy)
		}

		direction := "ASC"
		if sortDesc {
			direction = "DESC"
		}

		sortBy = fmt.Sprintf(" ORDER BY %v.%v %v", mdl.tableName, sortCol, direction)
	}

	fromWhere := fmt.Sprintf("FROM %v %v", sqlTables, whereStr)
	rowQuery := fmt.Sprintf("SELECT %v %v%v LIMIT %v,%v", sqlColumns, fromWhere, sortBy, page*25, 25)
	countQuery := fmt.Sprintf("SELECT COUNT(*) %v", fromWhere)
	fmt.Println(rowQuery)

	var rows *sql.Rows
	var countRow *sql.Row
	var err error
	if doSearch {
		rows, err = a.db.Query(rowQuery, searchList...)
		countRow = a.db.QueryRow(countQuery, searchList...)
	} else {
		rows, err = a.db.Query(rowQuery)
		countRow = a.db.QueryRow(countQuery)
	}

	numRows := 0
	err = countRow.Scan(&numRows)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(numRows)

	if err != nil {
		return nil, numRows, err
	}

	numCols := len(cols)
	results := [][]interface{}{}

	for rows.Next() {
		result, err := scanRow(numCols, rows)
		if err != nil {
			return nil, numRows, err
		}
		results = append(results, result)
	}

	return results, numRows, nil
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
