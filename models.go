package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/extemporalgenome/slug"
	"github.com/oal/admin/db"
	"github.com/oal/admin/fields"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

// NamedModel requires an AdminName method to be present, to override the model's displayed name in the admin panel.
type NamedModel interface {
	AdminName() string
}

type SortedModel interface {
	SortBy() string
}

type modelGroup struct {
	admin  *Admin
	Name   string
	slug   string
	Models []*model
}

// RegisterModel adds a model to a model group.
func (g *modelGroup) RegisterModel(mdl interface{}) error {
	modelType := reflect.TypeOf(mdl)
	ind := reflect.Indirect(reflect.ValueOf(mdl))

	name := typeToName(modelType)
	tableName := typeToTableName(modelType, g.admin.NameTransform)

	if named, ok := mdl.(NamedModel); ok {
		name = named.AdminName()
	}

	newModel := model{
		Name:      name,
		Slug:      slug.SlugAscii(name),
		tableName: tableName,
		fields:    []fields.Field{},

		fieldNames:        []string{},
		listFields:        []fields.Field{},
		searchableColumns: []string{},

		admin: g.admin,
	}

	// Set as registered so it can be used as a ForeignKey from other models
	if _, ok := g.admin.registeredRels[modelType]; !ok {
		g.admin.registeredRels[modelType] = &newModel
	}

	// Check if any fields previously registered is missing this model as a foreign key
	for field, missingType := range g.admin.missingRels {
		if missingType != modelType {
			continue
		}

		field.SetModelSlug(newModel.Slug)
		delete(g.admin.missingRels, field)
	}

	// Loop over struct fields and set up fields
	for i := 0; i < ind.NumField(); i++ {
		refl := modelType.Elem().Field(i)
		fieldType := refl.Type
		kind := fieldType.Kind()

		// Expect pointers to be foreign keys and foreign keys to have the form Field[Id]
		fieldName := refl.Name
		if kind == reflect.Ptr {
			fieldName += "Id"
		}

		// Parse key=val / key options from struct tag, used for configuration later
		tag := refl.Tag.Get("admin")
		if tag == "-" {
			if i == 0 {
				return errors.New("First column (id) can't be skipped.")
			}
			continue
		}
		tagMap, err := parseTag(tag)
		if err != nil {
			panic(err)
		}

		// ID (i == 0) is always shown
		if i == 0 {
			tagMap["list"] = ""
		}

		override, _ := tagMap["field"]
		field := makeField(kind, override)

		// If slice, get type / kind of elements instead
		// makeField still needs to know it's a slice, but this is needed below
		if kind == reflect.Slice {
			fieldType = fieldType.Elem()
			kind = fieldType.Kind()
		}

		// Relationships need some additional data added to them
		if relField, ok := field.(fields.RelationalField); ok {
			// If column is shown in list view, and a field in related model is set to be listed
			if listField, ok := tagMap["list"]; ok && len(listField) != 0 {
				if g.admin.NameTransform != nil {
					listField = g.admin.NameTransform(listField)
				}
				relField.SetListColumn(listField)
			}

			relField.SetRelatedTable(typeToTableName(fieldType, g.admin.NameTransform))

			// We also need the field to know what model it's related to
			if regModel, ok := g.admin.registeredRels[fieldType]; ok {
				relField.SetModelSlug(regModel.Slug)
			} else {
				g.admin.missingRels[relField] = fieldType
			}
			field, _ = relField.(fields.Field)
		}

		// Transform struct keys to DB column names if needed
		var tableField string
		if g.admin.NameTransform != nil {
			tableField = g.admin.NameTransform(fieldName)
		} else {
			tableField = refl.Name
		}

		field.Attrs().Name = fieldName
		field.Attrs().ColumnName = tableField
		applyFieldTags(&newModel, field, tagMap)

		newModel.fields = append(newModel.fields, field)
		newModel.fieldNames = append(newModel.fieldNames, fieldName)
	}

	// Default sorting in list view
	if sorted, ok := mdl.(SortedModel); ok && newModel.fieldByName(sorted.SortBy()) != nil {
		newModel.sort = sorted.SortBy()
	} else {
		newModel.sort = "-Id"
	}

	g.admin.models[newModel.Slug] = &newModel
	g.Models = append(g.Models, &newModel)

	fmt.Println("Registered", newModel.Name)
	return nil
}

func makeField(kind reflect.Kind, override string) fields.Field {
	// First, check if we want to override a field, otherwise use one of the defaults
	var field fields.Field
	if customField := fields.GetCustom(override); customField != nil {
		// Create field
		customType := reflect.ValueOf(customField).Elem().Type()
		newField := reflect.New(customType)

		// Create BaseField in Field
		baseField := newField.Elem().Field(0)
		baseField.Set(reflect.ValueOf(&fields.BaseField{}))

		field = newField.Interface().(fields.Field)
	} else {
		switch kind {
		case reflect.String:
			field = &fields.TextField{BaseField: &fields.BaseField{}}
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field = &fields.IntField{BaseField: &fields.BaseField{}}
		case reflect.Float32, reflect.Float64:
			field = &fields.FloatField{BaseField: &fields.BaseField{}}
		case reflect.Bool:
			field = &fields.BooleanField{BaseField: &fields.BaseField{}}
		case reflect.Struct:
			field = &fields.TimeField{BaseField: &fields.BaseField{}}
		case reflect.Ptr:
			field = &fields.ForeignKeyField{BaseField: &fields.BaseField{}}
		case reflect.Slice:
			field = &fields.ManyToManyField{BaseField: &fields.BaseField{}}
		default:
			fmt.Println("Unknown field type")
			field = &fields.TextField{BaseField: &fields.BaseField{}}
		}
	}

	return field
}

func applyFieldTags(mdl *model, field fields.Field, tagMap map[string]string) {
	// Read relevant config options from the tagMap
	err := field.Configure(tagMap)
	if err != nil {
		panic(err)
	}

	if label, ok := tagMap["label"]; ok {
		field.Attrs().Label = label
	} else {
		field.Attrs().Label = field.Attrs().Name
	}

	if _, ok := tagMap["blank"]; ok {
		field.Attrs().Blank = true
	}

	if _, ok := tagMap["null"]; ok {
		field.Attrs().Null = true
	}

	if _, ok := tagMap["list"]; ok {
		field.Attrs().List = true
		mdl.listFields = append(mdl.listFields, field)
	}

	if _, ok := tagMap["search"]; ok {
		field.Attrs().Searchable = true
		mdl.searchableColumns = append(mdl.searchableColumns, field.Attrs().ColumnName)
	}

	if val, ok := tagMap["default"]; ok {
		field.Attrs().DefaultValue = val
	}

	if width, ok := tagMap["width"]; ok {
		i, err := parseInt(width)
		if err != nil {
			panic(err)
		}
		field.Attrs().Width = i
	}
}

type model struct {
	Name      string
	Slug      string
	fields    []fields.Field
	tableName string

	fieldNames        []string
	listFields        []fields.Field
	searchableColumns []string
	sort              string

	admin *Admin
}

func (m *model) renderForm(w io.Writer, data map[string]interface{}, defaults bool, errors map[string]string) {
	var val interface{}
	var ok bool
	activeCol := 0
	for _, fieldName := range m.fieldNames[1:] {
		field := m.fieldByName(fieldName)
		val, ok = data[fieldName]
		if !ok && defaults {
			val = field.Attrs().DefaultValue
		}

		// Error text displayed below field, if any
		var err string
		if errors != nil {
			err = errors[fieldName]
		}

		field.Render(w, val, err, activeCol%12 == 0)
		activeCol += field.Attrs().Width
	}
}

func (m *model) fieldByName(name string) fields.Field {
	for _, field := range m.fields {
		if field.Attrs().Name == name {
			return field
		}
	}
	return nil
}

func (m *model) get(id int) (map[string]interface{}, error) {
	cols := make([]string, 0, len(m.fieldNames))
	m2mFields := map[string]struct{}{}

	// Can't do * as column order in the DB might not match struct
	for _, fieldName := range m.fieldNames {
		// Add to m2mFields so we can load it later
		if _, ok := m.fieldByName(fieldName).(*fields.ManyToManyField); ok {
			m2mFields[fieldName] = struct{}{}
			continue
		}

		// Normal columns will be loaded directly in the main query
		if m.admin.NameTransform != nil {
			fieldName = m.admin.NameTransform(fieldName)
		}
		cols = append(cols, fieldName)
	}

	q := m.admin.dialect.Queryf("SELECT %v FROM %v WHERE id = ?", strings.Join(cols, ", "), m.tableName)
	row := m.admin.db.QueryRow(q, id)

	result, err := db.ScanRow(len(cols), row)
	if err != nil {
		return nil, err
	}

	// Loop over fields, and run separate query for M2Ms. Iterator index i only increases if there is value for column in main query.
	resultMap := map[string]interface{}{}
	i := 0
	for _, fieldName := range m.fieldNames {
		if _, ok := m2mFields[fieldName]; ok {
			// Get Id of all related rows
			field, ok := m.fieldByName(fieldName).(*fields.ManyToManyField)
			if !ok {
				continue
			}
			relTable := field.GetRelatedTable()

			q := m.admin.dialect.Queryf("SELECT %v_id FROM %v_%v WHERE %v_id = ?", relTable, m.tableName, field.ColumnName, m.tableName)
			rows, err := m.admin.db.Query(q, id)
			if err != nil {
				return nil, err
			}

			ids := []int{}
			for rows.Next() {
				var relId int
				rows.Scan(&relId)
				ids = append(ids, relId)
			}

			resultMap[fieldName] = ids
			continue
		}

		// Any other data type
		resultMap[fieldName] = result[i]
		i++
	}

	return resultMap, nil
}

func (m *model) page(page int, search, sortBy string, sortDesc bool) ([][]interface{}, int, error) {
	page--

	// Ugly search. Will fix later.
	doSearch := false
	whereStr := ""
	var searchList []interface{}
	if len(search) > 0 {
		if len(m.searchableColumns) > 0 {
			searchCols := make([]string, len(m.searchableColumns))
			searchList = make([]interface{}, len(searchCols))
			for i, _ := range searchList {
				searchList[i] = search
			}
			for i, _ := range searchCols {
				searchCols[i] = fmt.Sprintf("%v.%v LIKE ?", m.tableName, m.searchableColumns[i])
			}
			whereStr = fmt.Sprintf("WHERE (%v)", strings.Join(searchCols, " OR "))
			doSearch = true
		}

	}

	cols := []string{}
	tables := []string{m.tableName}
	fkWhere := []string{}
	for _, field := range m.fields {
		if field.Attrs().List {
			colName := fmt.Sprintf("%v.%v", m.tableName, field.Attrs().ColumnName)
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
		if m.admin.NameTransform != nil {
			sortCol = m.admin.NameTransform(sortBy)
		}

		direction := "ASC"
		if sortDesc {
			direction = "DESC"
		}

		sortBy = fmt.Sprintf(" ORDER BY %v.%v %v", m.tableName, sortCol, direction)
	}

	fromWhere := fmt.Sprintf("FROM %v %v", sqlTables, whereStr)
	rowQuery := m.admin.dialect.Queryf("SELECT %v %v%v LIMIT %v,%v", sqlColumns, fromWhere, sortBy, page*25, 25)
	countQuery := m.admin.dialect.Queryf("SELECT COUNT(*) %v", fromWhere)

	var rows *sql.Rows
	var countRow *sql.Row
	var err error
	if doSearch {
		rows, err = m.admin.db.Query(rowQuery, searchList...)
		countRow = m.admin.db.QueryRow(countQuery, searchList...)
	} else {
		rows, err = m.admin.db.Query(rowQuery)
		countRow = m.admin.db.QueryRow(countQuery)
	}

	numRows := 0
	err = countRow.Scan(&numRows)
	if err != nil {
		fmt.Println(err)
	}

	if err != nil {
		return nil, numRows, err
	}

	numCols := len(cols)
	results := [][]interface{}{}

	for rows.Next() {
		result, err := db.ScanRow(numCols, rows)
		if err != nil {
			return nil, numRows, err
		}
		results = append(results, result)
	}

	return results, numRows, nil
}

func (m *model) save(id int, req *http.Request) (map[string]interface{}, map[string]string, error) {
	numFields := len(m.fieldNames) - 1 // No need for ID.

	// Get existing data, if any, so we can check what values were changed (existing == nil for new rows)
	var existing map[string]interface{}
	if id != 0 {
		var err error
		existing, err = m.get(id)
		if err != nil {
			return nil, nil, err
		}
	}

	// Get data from POST and fill a slice
	data := map[string]interface{}{}
	m2mData := map[string][]int{}
	m2mChanges := false
	dataErrors := map[string]string{}
	hasErrors := false
	for i := 0; i < numFields; i++ {
		fieldName := m.fieldNames[i+1]
		field := m.fieldByName(fieldName)

		var existingVal interface{}
		if existing != nil {
			existingVal = existing[fieldName]
		}
		val, err := fields.Validate(field, req, existingVal)
		if err != nil {
			dataErrors[fieldName] = err.Error()
			hasErrors = true
		}

		// ManyToManyField
		if ids, ok := val.([]int); ok {
			// Has M2M data changed?
			if existingIds, ok := existingVal.([]int); ok {
				// FIXME: Maybe not the best way to compare slices?
				sort.Ints(ids)
				sort.Ints(existingIds)
				m2mChanges = fmt.Sprint(ids) != fmt.Sprint(existingIds)
			} else if len(ids) > 0 {
				m2mChanges = true
			}

			m2mData[fieldName] = ids
			continue
		}

		data[fieldName] = val
	}

	if hasErrors {
		return data, dataErrors, errors.New("Please correct the errors below.")
	}

	// Create query only with the changed data
	changedCols := []string{}
	changedData := []interface{}{}
	for key, value := range data {
		// Skip if not changed
		if existing != nil && value == existing[key] {
			continue
		}

		// Convert to DB version of name and append
		col := key
		if m.admin.NameTransform != nil {
			col = m.admin.NameTransform(key)
		}
		if id != 0 {
			col = fmt.Sprintf("%v = ?", col)
		}
		changedCols = append(changedCols, col)
		changedData = append(changedData, value)
	}

	if len(changedCols) == 0 && !m2mChanges {
		return nil, nil, errors.New(fmt.Sprintf("%v was not saved because there were no changes.", m.Name))
	}

	if len(changedCols) > 0 {
		valMarks := strings.Repeat("?, ", len(changedCols))
		valMarks = valMarks[0 : len(valMarks)-2]

		// Insert / update
		var q string
		if id != 0 {
			q = m.admin.dialect.Queryf("UPDATE %v SET %v WHERE id = %v", m.tableName, strings.Join(changedCols, ", "), id)
		} else {
			q = m.admin.dialect.Queryf("INSERT INTO %v(%v) VALUES(%v)", m.tableName, strings.Join(changedCols, ", "), valMarks)
		}

		result, err := m.admin.db.Exec(q, changedData...)
		if err != nil {
			fmt.Println(err)
			return nil, nil, err
		}

		if newId, _ := result.LastInsertId(); id == 0 {
			id = int(newId)
		}
	}

	if m2mChanges {
		// Insert / update M2M
		for fieldName, ids := range m2mData {
			field, _ := m.fieldByName(fieldName).(*fields.ManyToManyField)
			err := m.saveM2M(id, field, ids)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return data, dataErrors, nil
}

func (m *model) saveM2M(id int, field *fields.ManyToManyField, relatedIds []int) error {
	m2mTable := fmt.Sprintf("%v_%v", m.tableName, field.ColumnName)
	toColumn := fmt.Sprintf("%v_id", field.GetRelatedTable())
	fromColumn := fmt.Sprintf("%v_id", m.tableName)

	tx, err := m.admin.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Commit()

	existingRelQuery := m.admin.dialect.Queryf("SELECT %v FROM %v WHERE %v = ?", toColumn, m2mTable, fromColumn)
	rows, err := tx.Query(existingRelQuery, id)
	if err != nil {
		return err
	}

	removeRels := map[int]bool{}
	for rows.Next() {
		var eId int
		rows.Scan(&eId)
		removeRels[eId] = true
	}

	// Add new, remove from removeRels as we go. Those still left in removeRels will be deleted.
	for _, nId := range relatedIds {
		if _, ok := removeRels[nId]; ok {
			// Already exists,
			delete(removeRels, nId)
		} else {
			// Relation doesn't exist yet, so add it
			addRelQuery := m.admin.dialect.Queryf("INSERT INTO %v (%v, %v) VALUES (?, ?)", m2mTable, fromColumn, toColumn)
			tx.Exec(addRelQuery, id, nId)
		}
	}

	// Delete remaining Ids in removeRels as they're no longer related
	for eId, _ := range removeRels {
		removeRelQuery := m.admin.dialect.Queryf("DELETE FROM %v WHERE %v = ? AND %v = ?", m2mTable, fromColumn, toColumn)
		tx.Exec(removeRelQuery, id, eId)
	}

	return nil
}

func (m *model) delete(id int) error {
	_, err := m.get(id)
	if err != nil {
		return err
	}

	q := fmt.Sprintf("DELETE FROM %v WHERE id=?", m.tableName)
	_, err = m.admin.db.Exec(q, id)
	if err != nil {
		return err
	}

	// Delete M2M relations
	for _, fieldName := range m.fieldNames {
		if field, ok := m.fieldByName(fieldName).(*fields.ManyToManyField); ok {
			m2mTable := fmt.Sprintf("%v_%v", m.tableName, field.ColumnName)
			fromColumn := fmt.Sprintf("%v_id", m.tableName)
			q := m.admin.dialect.Queryf("DELETE FROM %v WHERE %v = ?", m2mTable, fromColumn)
			m.admin.db.Exec(q, id)
		}
	}

	return nil
}
