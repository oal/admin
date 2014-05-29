package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/extemporalgenome/slug"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"reflect"
	"strings"
)

var templates, _ = template.ParseGlob(
	"admin/templates/*.html",
)

type Admin struct {
	Router        *mux.Router
	Path          string
	Database      string
	Title         string
	NameTransform func(string) string
	db            *sql.DB
	models        map[string]*model
	modelGroups   []*modelGroup
}

func (a *Admin) Serve() error {
	if len(a.Title) == 0 {
		a.Title = "Admin"
	}

	db, err := sql.Open("sqlite3", a.Database)
	if err != nil {
		return err
	}
	a.db = db
	fmt.Println("DB loaded")

	a.models = map[string]*model{}
	a.modelGroups = []*modelGroup{}

	sr := a.Router.PathPrefix(a.Path).Subrouter()
	sr.StrictSlash(true)
	sr.HandleFunc("/", a.handleIndex)
	sr.HandleFunc("/model/{slug}/", a.handleList)
	sr.HandleFunc("/model/{slug}/new/", a.handleEdit)
	sr.HandleFunc("/model/{slug}/edit/{id}/", a.handleEdit)
	return nil
}

func (a *Admin) Group(name string) (*modelGroup, error) {
	if a.models == nil {
		return nil, errors.New("Must call .Serve() before adding groups and registering models")
	}

	group := &modelGroup{
		admin:  a,
		Name:   name,
		slug:   slug.SlugAscii(name),
		Models: []*model{},
	}

	a.modelGroups = append(a.modelGroups, group)

	return group, nil
}

type modelGroup struct {
	admin  *Admin
	Name   string
	slug   string
	Models []*model
}

func (g *modelGroup) RegisterModel(mdl interface{}) error {
	t := reflect.TypeOf(mdl)

	val := reflect.ValueOf(mdl)
	ind := reflect.Indirect(val)

	parts := strings.Split(t.String(), ".")
	name := parts[len(parts)-1]
	var tableName string
	if g.admin.NameTransform != nil {
		tableName = g.admin.NameTransform(name)
	} else {
		tableName = name
	}
	am := model{
		Name:      name,
		Slug:      slug.SlugAscii(name),
		tableName: tableName,
		fields:    []*modelField{},
		instance:  mdl,
	}

	for i := 0; i < ind.NumField(); i++ {
		field := t.Elem().Field(i)
		tag := field.Tag.Get("admin")
		if tag == "-" {
			continue
		}

		fieldName := field.Name
		if len(field.Type.Name()) == 0 {
			fieldName += "Id"
		}

		var tableField string
		if g.admin.NameTransform != nil {
			tableField = g.admin.NameTransform(fieldName)
		} else {
			tableField = field.Name
		}

		modelField := &modelField{
			name:       fieldName,
			columnName: tableField,
			field:      &TextField{Name: fieldName},
		}

		tagVals := strings.Split(tag, ",")
		for _, tag := range tagVals {
			switch tag {
			case "list":
				modelField.list = true
			}
		}

		am.fields = append(am.fields, modelField)
	}

	g.admin.models[am.Slug] = &am
	g.Models = append(g.Models, &am)
	return nil
}

type model struct {
	Name      string
	Slug      string
	fields    []*modelField
	tableName string
	instance  interface{}
}

type modelField struct {
	name       string
	columnName string
	list       bool
	field      Field
}

func (m *model) fieldNames() []string {
	names := []string{}
	for _, field := range m.fields {
		names = append(names, field.name)
	}
	return names
}

func (m *model) tableColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		names = append(names, field.columnName)
	}
	return names
}

func (m *model) listColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		if !field.list {
			continue
		}
		names = append(names, field.name)
	}
	return names
}

func (m *model) listTableColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		if !field.list {
			continue
		}
		names = append(names, field.columnName)
	}
	return names
}

func (m *model) fieldByName(name string) Field {
	for _, field := range m.fields {
		if field.name == name {
			return field.field
		}
	}
	return nil
}

func (m *model) render(w io.Writer, data []string) {
	hasData := len(data) == len(m.fieldNames())
	var val string
	for i, fieldName := range m.fieldNames() {
		if hasData {
			val = data[i]
		}
		m.fieldByName(fieldName).Render(w, val)
	}
}

func (a *Admin) modelURL(slug, action string) string {
	if _, ok := a.models[slug]; !ok {
		return a.Path
	}

	return fmt.Sprintf("%v/model/%v%v", a.Path, slug, action)
}

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

func (a *Admin) querySingleModel(mdl *model, id int) ([]string, error) {
	numCols := len(mdl.fieldNames()) + 1
	rawResult := make([][]byte, numCols)
	dest := make([]interface{}, numCols)
	for i, _ := range rawResult {
		dest[i] = &rawResult[i]
	}

	q := fmt.Sprintf("SELECT id, %v FROM %v WHERE id = ?", strings.Join(mdl.tableColumns(), ","), mdl.tableName)
	err := a.db.QueryRow(q, id).Scan(dest...)
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

func scanRow(numCols int, rows *sql.Rows) ([]string, error) {
	rawResult := make([][]byte, numCols)
	dest := make([]interface{}, numCols)
	for i, _ := range rawResult {
		dest[i] = &rawResult[i]
	}

	err := rows.Scan(dest...)
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
