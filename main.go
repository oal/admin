package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"reflect"

	"github.com/extemporalgenome/slug"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oal/admin/db"
	"github.com/oal/admin/fields"
)

// NameTransformFunc is a function that takes the name of a Go struct field and outputs another version of itself.
// This is used to be compatible with various ORMs. See NameTransform on the Admin struct.
type NameTransformFunc func(string) string

type Admin struct {
	// Title allows you to set a custom title for the admin panel. Default is "Admin".
	Title string

	// NameTransform is optional, and allows you to set a function that model names and field names are sent through
	// to maintain compatibility with an ORM. For example, Beego ORM saves tables/columns in snake_case, while CamelCase
	// is used in Go.
	NameTransform NameTransformFunc

	path      string
	username  string
	password  string
	sessions  map[string]*session
	urls      *urlConfig
	db        *sql.DB
	dialect   db.Dialect
	sourceDir string

	models         map[string]*model
	modelGroups    []*modelGroup
	registeredRels map[reflect.Type]*model
	missingRels    map[fields.RelationalField]reflect.Type
}

// New sets up the admin with a "path" prefix (typically /admin) and the name of a database driver and source.
func New(path, dbDriver, dbSource string) (*Admin, error) {
	admin := &Admin{}

	err := admin.database(dbDriver, dbSource)
	if err != nil {
		return nil, err
	}

	admin.sourceDir = fmt.Sprintf("%v/src/github.com/oal/admin", os.Getenv("GOPATH"))
	admin.path = path
	admin.Title = "Admin"

	admin.sessions = map[string]*session{}

	// Model init
	admin.models = map[string]*model{}
	admin.modelGroups = []*modelGroup{}
	admin.registeredRels = map[reflect.Type]*model{}
	admin.missingRels = map[fields.RelationalField]reflect.Type{}

	return admin, nil
}

// SourceDir allows you to override the location in which templates and static content is looked for / served from.
// If not set, it defaults to $GOPATH/src/github.com/oal/admin. You may also copy "templates" and "static" from there,
// into your own project, and change SourceDir accordingly.
func (a *Admin) SourceDir(dir string) error {
	if _, err := os.Stat(fmt.Sprintf("%v/templates/", dir)); err != nil {
		return err
	}

	a.sourceDir = dir
	return nil
}

// User sets username and password for the admin panel. This will change in the future, when support for custom login backends
// is implemented. No promises on when that will happen, though.
func (a *Admin) User(username, password string) error {
	if len(username) == 0 || len(password) == 0 {
		return errors.New("Username and/or password is missing")
	}

	a.username = username
	a.password = password

	return nil
}

// Handler returns a http.Handler that you can attach to any mux to serve the admin.
func (a *Admin) Handler() (http.Handler, error) {
	staticDir := fmt.Sprintf("%v/static/", a.sourceDir)
	if _, err := os.Stat(staticDir); err != nil {
		return nil, err
	}

	// Load templates (only once, in case we run multiple admins)
	if templates == nil {
		var err error
		templates, err = template.New("admin").Funcs(template.FuncMap{
			"url": func(name string, args ...interface{}) string {
				url, err := a.urls.URL(name, args...)
				if err != nil {
					fmt.Println(err)
				}
				return url
			},
		}).ParseGlob(fmt.Sprintf("%v/templates/*.html", a.sourceDir))
		if err != nil {
			panic(err)
		}
	}

	urls := newURLConfig(a.path)
	urls.router.RedirectTrailingSlash = true
	urls.router.RedirectFixedPath = true

	urls.add("index", "GET", "/", a.handlerWrapper(a.handleIndex))
	urls.add("login", "POST", "/", a.handlerWrapper(a.handleIndex))
	urls.add("logout", "GET", "/logout/", a.handlerWrapper(a.handleLogout))

	urls.add("view", "GET", "/view/:slug/", a.handlerWrapper(a.handleList))
	urls.add("view2", "GET", "/view/:slug/:view/*multiselect", a.handlerWrapper(a.handleList))

	urls.add("new", "GET", "/new/:slug/", a.handlerWrapper(a.handleEdit))
	urls.add("create", "POST", "/create/:slug/", a.handlerWrapper(a.handleEdit))

	urls.add("edit", "GET", "/edit/:slug/:id/", a.handlerWrapper(a.handleEdit))
	urls.add("save", "POST", "/save/:slug/:id/", a.handlerWrapper(a.handleEdit))

	urls.add("delete", "GET", "/delete/:slug/:id/", a.handlerWrapper(a.handleDelete))

	urls.router.ServeFiles(a.path+"/static/*filepath", http.Dir(staticDir))

	a.urls = urls
	return urls.router, nil
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

func (a *Admin) database(driver, source string) error {
	adminDB, err := sql.Open(driver, source)
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

	a.db = adminDB
	return nil
}
