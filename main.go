package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/extemporalgenome/slug"
	"github.com/julienschmidt/httprouter"
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
	Path     string
	Username string
	Password string

	sessions map[string]*session

	SourceDir string

	title         string
	nameTransform NameTransformFunc

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

	admin.title = "Admin"

	// Users / sessions
	if len(admin.Username) == 0 || len(admin.Password) == 0 {
		return nil, errors.New("Username and/or password is missing")
	}
	admin.sessions = map[string]*session{}

	if _, err := os.Stat(fmt.Sprintf("%v/templates/", admin.SourceDir)); err != nil {
		return nil, err
	}

	// Model init
	admin.models = map[string]*model{}
	admin.modelGroups = []*modelGroup{}
	admin.registeredRels = map[reflect.Type]*model{}
	admin.missingRels = map[fields.RelationalField]reflect.Type{}

	// Routes

	return admin, nil
}

// SetTitle allows you to change the page title for your admin panel.
func (a *Admin) SetTitle(title string) {
	a.title = title
}

// SetDatabase sets the database the admin connects to.
func (a *Admin) SetDatabase(driver, source string) error {
	database, err := sql.Open(driver, source)
	if err != nil {
		return err
	}

	switch driver {
	case "postgres":
		a.dialect = db.PostgresDialect{}
	case "sqlite3", "mysql":
		a.dialect = db.BaseDialect{}
	default:
		return errors.New(fmt.Sprintf("Unknown database driver %v", driver))
	}

	a.db = database
	return nil
}

// SetNameTransformer is optional, and allows you to set a function that model names and field names are sent through
// to maintain compatibility with an ORM. For example, Beego ORM saves tables/columns in snake_case, while CamelCase
// is used in Go.
func (a *Admin) SetNameTransformer(nameFunc NameTransformFunc) {
	a.nameTransform = nameFunc
}

// Handler returns a http.Handler that you can attach to any mux to serve the admin.
func (a *Admin) Handler() (http.Handler, error) {
	staticDir := fmt.Sprintf("%v/static/", a.SourceDir)
	if _, err := os.Stat(staticDir); err != nil {
		return nil, err
	}

	r := httprouter.New()
	r.RedirectTrailingSlash = true
	r.RedirectFixedPath = true

	r.GET(a.Path+"/", a.handlerWrapper(a.handleIndex))
	r.POST(a.Path+"/", a.handlerWrapper(a.handleIndex))
	r.GET(a.Path+"/logout/", a.handlerWrapper(a.handleLogout))

	r.GET(a.Path+"/view/:slug/", a.handlerWrapper(a.handleList))
	r.GET(a.Path+"/view/:slug/:view/", a.handlerWrapper(a.handleList))
	r.GET(a.Path+"/new/:slug/", a.handlerWrapper(a.handleEdit))

	r.GET(a.Path+"/edit/:slug/:id/", a.handlerWrapper(a.handleEdit))
	r.POST(a.Path+"/edit/:slug/:id/", a.handlerWrapper(a.handleEdit))

	r.GET(a.Path+"/delete/:slug/:id/", a.handlerWrapper(a.handleDelete))
	r.ServeFiles(a.Path+"/static/*filepath", http.Dir(staticDir))

	return r, nil
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

func (a *Admin) modelURL(slug, action string, id int) string {
	if _, ok := a.models[slug]; !ok {
		return a.Path
	}

	// Improve this
	if action == "view" {
		return fmt.Sprintf("%v/%v/%v/", a.Path, action, slug)
	}

	return fmt.Sprintf("%v/%v/%v/%v/", a.Path, action, slug, id)
}

func loadTemplates(path string) (*template.Template, error) {
	// Pages / views
	tmpl, err := template.ParseGlob(fmt.Sprintf("%v/*.html", path))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}
