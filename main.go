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
	Username      string
	Password      string
	db            *sql.DB
	models        map[string]*model
	modelGroups   []*modelGroup
}

func Setup(admin *Admin) (*Admin, error) {
	if len(admin.Title) == 0 {
		admin.Title = "Admin"
	}

	if len(admin.Username) == 0 || len(admin.Password) == 0 {
		return nil, errors.New("Username and/or password is missing")
	}

	db, err := sql.Open("sqlite3", admin.Database)
	if err != nil {
		return nil, err
	}
	admin.db = db

	admin.models = map[string]*model{}
	admin.modelGroups = []*modelGroup{}

	sr := admin.Router.PathPrefix(admin.Path).Subrouter()
	sr.StrictSlash(true)
	sr.HandleFunc("/", admin.handleIndex)
	sr.HandleFunc("/model/{slug}/", admin.handleList)
	sr.HandleFunc("/model/{slug}/new/", admin.handleEdit)
	sr.HandleFunc("/model/{slug}/edit/{id}/", admin.handleEdit)
	return admin, nil
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
		fieldType := field.Type
		kind := fieldType.Kind()
		tag := field.Tag.Get("admin")
		if tag == "-" {
			continue
		}

		// Parse key=val / key options from struct tag, used for configuration later
		tagMap, err := parseTag(tag)
		if err != nil {
			panic(err)
		}

		// Expect pointers to be foreignkeys and foreignkeys to have the form Field[Id]
		fieldName := field.Name
		if kind == reflect.Ptr {
			fieldName += "Id"
		}

		// Transform struct keys to DB column names if needed
		var tableField string
		if g.admin.NameTransform != nil {
			tableField = g.admin.NameTransform(fieldName)
		} else {
			tableField = field.Name
		}

		var widget Widget
		fmt.Println(kind)
		if widgetType, ok := tagMap["widget"]; ok {
			switch widgetType {
			case "url":
				widget = &URLWidget{}
			default:
				widget = &TextWidget{}
			}
		} else {
			switch kind {
			case reflect.String:
				widget = &TextWidget{}
			case reflect.Int:
				widget = &NumberWidget{}
			case reflect.Struct:
				widget = &TimeWidget{}
			default:
				fmt.Println("NOOO")
				widget = &TextWidget{}
			}
		}

		// Auto find widget

		// Read relevant config options from the tagMap
		err = widget.Configure(tagMap)
		if err != nil {
			panic(err)
		}

		modelField := &modelField{
			name:       fieldName,
			columnName: tableField,
			field:      widget,
		}

		if _, ok := tagMap["list"]; ok {
			modelField.list = true
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

func (m *model) renderForm(w io.Writer, data []interface{}) {
	hasData := len(data) == len(m.fieldNames())
	var val interface{}
	for i, fieldName := range m.fieldNames() {
		if hasData {
			val = data[i]
		}
		field := m.fieldByName(fieldName)
		field.field.Render(w, field.name, val)
	}
}

type modelField struct {
	name       string
	columnName string
	list       bool
	field      Widget
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

func (m *model) fieldByName(name string) *modelField {
	for _, field := range m.fields {
		if field.name == name {
			return field
		}
	}
	return nil
}

func (a *Admin) modelURL(slug, action string) string {
	if _, ok := a.models[slug]; !ok {
		return a.Path
	}

	return fmt.Sprintf("%v/model/%v%v", a.Path, slug, action)
}
