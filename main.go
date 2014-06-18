package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/extemporalgenome/slug"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oal/admin/db"
	"github.com/oal/admin/fields"
	"html/template"
	"net/http"
	"os"
	"reflect"
)

type NameTransformFunc func(string) string

type Admin struct {
	Router        *mux.Router
	Path          string
	Database      string
	Title         string
	NameTransform NameTransformFunc

	Username string
	Password string
	sessions map[string]*session

	SourceDir string

	db      *sql.DB
	dialect db.Dialect

	models         map[string]*model
	modelGroups    []*modelGroup
	registeredRels map[reflect.Type]*model
	missingRels    map[fields.RelationalField]reflect.Type
}

// Setup registers page handlers and enables the admin.
func Setup(admin *Admin) (*Admin, error) {
	// Source dir / static / templates
	if len(admin.SourceDir) == 0 {
		admin.SourceDir = fmt.Sprintf("%v/src/github.com/oal/admin", os.Getenv("GOPATH"))
	}

	// Load templates (only once, in case we run multiple admins)
	if templates == nil {
		var err error
		templates, err = loadTemplates(fmt.Sprintf("%v/templates", admin.SourceDir))
		if err != nil {
			panic(err)
		}
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
	database, err := sql.Open("sqlite3", admin.Database)
	if err != nil {
		return nil, err
	}
	admin.db = database

	admin.dialect = db.BaseDialect{}

	// Model init
	admin.models = map[string]*model{}
	admin.modelGroups = []*modelGroup{}
	admin.registeredRels = map[reflect.Type]*model{}
	admin.missingRels = map[fields.RelationalField]reflect.Type{}

	// Routes
	sr := admin.Router.PathPrefix(admin.Path).Subrouter()
	sr.StrictSlash(true)
	sr.HandleFunc("/", admin.handlerWrapper(admin.handleIndex))
	sr.HandleFunc("/logout/", admin.handlerWrapper(admin.handleLogout))
	sr.HandleFunc("/model/{slug}/", admin.handlerWrapper(admin.handleList))
	sr.HandleFunc("/model/{slug}/new/", admin.handlerWrapper(admin.handleEdit))
	sr.HandleFunc("/model/{slug}/{view}/", admin.handlerWrapper(admin.handleList))
	sr.HandleFunc("/model/{slug}/edit/{id}/", admin.handlerWrapper(admin.handleEdit))
	sr.HandleFunc("/model/{slug}/delete/{id}/", admin.handlerWrapper(admin.handleDelete))
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

func (a *Admin) modelURL(slug, action string) string {
	if _, ok := a.models[slug]; !ok {
		return a.Path
	}

	return fmt.Sprintf("%v/model/%v%v", a.Path, slug, action)
}

func loadTemplates(path string) (*template.Template, error) {
	// Pages / views
	tmpl, err := template.ParseGlob(fmt.Sprintf("%v/*.html", path))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
