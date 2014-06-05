package admin

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/extemporalgenome/slug"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
)

type Admin struct {
	Router        *mux.Router
	Path          string
	Database      string
	Title         string
	NameTransform func(string) string

	Username string
	Password string
	sessions map[string]*session

	SourceDir string

	db            *sql.DB
	models        map[string]*model
	modelGroups   []*modelGroup
	registeredFKs map[reflect.Type]*model
	missingFKs    map[*ForeignKeyField]reflect.Type
}

// Setup registers page handlers and enables the admin.
func Setup(admin *Admin) (*Admin, error) {
	// Source dir / static / templates
	if len(admin.SourceDir) == 0 {
		admin.SourceDir = fmt.Sprintf("%v/src/github.com/oal/admin", os.Getenv("GOPATH"))
	}

	// Load templates (only once, in case we run multiple admins)
	if templates == nil {
		templates = template.Must(template.ParseGlob(
			fmt.Sprintf("%v/templates/*.html", admin.SourceDir),
		))
		fieldTemplates = template.Must(template.ParseGlob(
			fmt.Sprintf("%v/templates/fields/*.html", admin.SourceDir),
		))

		fieldWrapperTemplate = template.New("fieldWrapper")
		fieldWrapperTemplate.Funcs(template.FuncMap{
			"runtemplate": func(name string, ctx interface{}) (template.HTML, error) {
				var buf bytes.Buffer
				err := fieldTemplates.Lookup(name).Execute(&buf, ctx)
				if err != nil {
					return "", err
				}
				return template.HTML(buf.String()), nil
			},
		})
		fieldWrapperTemplate = template.Must(fieldWrapperTemplate.Parse(`
			<div class="form-group">
				<label for="{{.name}}">{{.label}}</label>
				{{runtemplate .tmpl .}}
				{{if .error}}<p class="text-danger">{{.error}}</p>{{end}}
			</div>
		`))

	}

	// Title
	if len(admin.Title) == 0 {
		admin.Title = "Admin"
	}

	// Users / sessions
	if len(admin.Username) == 0 || len(admin.Password) == 0 {
		return nil, errors.New("Username and/or password is missing")
	}
	admin.sessions = map[string]*session{}

	staticDir := fmt.Sprintf("%v/static/", admin.SourceDir)
	if _, err := os.Stat(staticDir); err != nil {
		return nil, err
	}
	if _, err := os.Stat(fmt.Sprintf("%v/templates/", admin.SourceDir)); err != nil {
		return nil, err
	}

	// Database
	db, err := sql.Open("sqlite3", admin.Database)
	if err != nil {
		return nil, err
	}
	admin.db = db

	// Model init
	admin.models = map[string]*model{}
	admin.modelGroups = []*modelGroup{}
	admin.registeredFKs = map[reflect.Type]*model{}
	admin.missingFKs = map[*ForeignKeyField]reflect.Type{}

	// Routes
	sr := admin.Router.PathPrefix(admin.Path).Subrouter()
	sr.StrictSlash(true)
	sr.HandleFunc("/", admin.handlerWrapper(admin.handleIndex))
	sr.HandleFunc("/logout/", admin.handlerWrapper(admin.handleLogout))
	sr.HandleFunc("/model/{slug}/", admin.handlerWrapper(admin.handleList))
	sr.HandleFunc("/model/{slug}/new/", admin.handlerWrapper(admin.handleEdit))
	sr.HandleFunc("/model/{slug}/{view}/", admin.handlerWrapper(admin.handleList))
	sr.HandleFunc("/model/{slug}/edit/{id}/", admin.handlerWrapper(admin.handleEdit))
	sr.PathPrefix("/static/").Handler(http.StripPrefix("/admin/static/", http.FileServer(http.Dir(staticDir))))

	return admin, nil
}

// Group adds a model group to the admin front page.
// Use this to organize your models.
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

type namedModel interface {
	AdminName() string
}

// RegisterModel adds a model to a model group.
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

	if named, ok := mdl.(namedModel); ok {
		name = named.AdminName()
	}

	am := model{
		Name:      name,
		Slug:      slug.SlugAscii(name),
		tableName: tableName,
		fields:    []Field{},
		instance:  mdl,
	}

	// Set as registered so it can be used as a ForeignKey from other models
	if _, ok := g.admin.registeredFKs[t]; !ok {
		g.admin.registeredFKs[t] = &am
	}

	// Check if any fields previously registered is missing this model as a foreign key
	for field, modelType := range g.admin.missingFKs {
		if modelType != t {
			continue
		}

		field.model = &am
		delete(g.admin.missingFKs, field)
	}

	// Loop over struct fields and set up fields
	for i := 0; i < ind.NumField(); i++ {
		refl := t.Elem().Field(i)
		fieldType := refl.Type
		kind := fieldType.Kind()
		tag := refl.Tag.Get("admin")
		if tag == "-" {
			continue
		}

		// Parse key=val / key options from struct tag, used for configuration later
		tagMap, err := parseTag(tag)
		if err != nil {
			panic(err)
		}

		// Expect pointers to be foreignkeys and foreignkeys to have the form Field[Id]
		fieldName := refl.Name
		if kind == reflect.Ptr {
			fieldName += "Id"
		}

		// Transform struct keys to DB column names if needed
		var tableField string
		if g.admin.NameTransform != nil {
			tableField = g.admin.NameTransform(fieldName)
		} else {
			tableField = refl.Name
		}

		// Choose Field
		var field Field
		fmt.Println(kind)
		if strType, ok := tagMap["field"]; ok {
			switch strType {
			case "url":
				field = &URLField{BaseField: &BaseField{}}
			default:
				field = &TextField{BaseField: &BaseField{}}
			}
		} else {
			switch kind {
			case reflect.String:
				field = &TextField{BaseField: &BaseField{}}
			case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				field = &IntField{BaseField: &BaseField{}}
			case reflect.Float32, reflect.Float64:
				field = &FloatField{BaseField: &BaseField{}}
			case reflect.Struct:
				field = &TimeField{BaseField: &BaseField{}}
			case reflect.Ptr:
				field = &ForeignKeyField{BaseField: &BaseField{}}

				// Special treatment for foreign keys
				// We need the field to know what model it's related to
				if regModel, ok := g.admin.registeredFKs[fieldType]; ok {
					field.(*ForeignKeyField).model = regModel
				} else {
					g.admin.missingFKs[field.(*ForeignKeyField)] = refl.Type
				}
			default:
				fmt.Println("Unknown field type")
				field = &TextField{BaseField: &BaseField{}}
			}
		}
		field.Attrs().name = fieldName

		// Read relevant config options from the tagMap
		err = field.Configure(tagMap)
		if err != nil {
			panic(err)
		}

		if label, ok := tagMap["label"]; ok {
			field.Attrs().label = label
		} else {
			field.Attrs().label = fieldName
		}

		field.Attrs().columnName = tableField

		if _, ok := tagMap["list"]; ok {
			field.Attrs().list = true
		}

		if _, ok := tagMap["search"]; ok {
			field.Attrs().searchable = true
		}

		am.fields = append(am.fields, field)
	}

	g.admin.models[am.Slug] = &am
	g.Models = append(g.Models, &am)

	fmt.Println("Registered", am.Name)
	return nil
}

type model struct {
	Name      string
	Slug      string
	fields    []Field
	tableName string
	instance  interface{}
}

func (m *model) renderForm(w io.Writer, data []interface{}, errors []string) {
	hasData := len(data) == len(m.fieldNames())
	var val interface{}
	for i, fieldName := range m.fieldNames() {
		if hasData {
			val = data[i]
		}
		var err string
		if errors != nil {
			err = errors[i]
		}
		field := m.fieldByName(fieldName)
		field.Render(w, val, err)
	}
}

func (m *model) fieldNames() []string {
	names := []string{}
	for _, field := range m.fields {
		names = append(names, field.Attrs().name)
	}
	return names
}

func (m *model) tableColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		names = append(names, field.Attrs().columnName)
	}
	return names
}

func (m *model) listColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		if !field.Attrs().list {
			continue
		}
		names = append(names, field.Attrs().label)
	}
	return names
}

func (m *model) listTableColumns() []string {
	names := []string{}
	for _, field := range m.fields {
		if !field.Attrs().list {
			continue
		}
		names = append(names, field.Attrs().columnName)
	}
	return names
}

func (m *model) searchableColumns() []string {
	cols := []string{}
	for _, field := range m.fields {
		if !field.Attrs().searchable {
			continue
		}
		cols = append(cols, field.Attrs().columnName)
	}
	return cols
}

func (m *model) fieldByName(name string) Field {
	for _, field := range m.fields {
		if field.Attrs().name == name {
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
