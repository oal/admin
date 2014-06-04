Admin
=====

Web based admin interface for Go, inspired by Django admin.

Currently only works with SQLite. WIP. See the example app in /example for a working test app.

Symlink / copy templates (they should be accessible from your-project/admin/templates)

	ln -s $GOPATH/src/github.com/oal/admin admin


Activate like this:

```go
router := mux.NewRouter()
a, err := admin.Setup(&admin.Admin{
	Title:         "My admin panel",
	Router:        router,
	Path:          "/admin",
	Database:      "db.sqlite",
	NameTransform: snakeString,

	Username: "admin",
	Password: "password"
})
if err != nil {
	panic(err)
}

g, err := a.Group("Main")
if err != nil {
	panic(err)
}
g.RegisterModel(new(Page))
```

A model is just a struct

```go
type Page struct {
	Id      int    `admin:"-"`
	Name    string `admin:"list"`
	Slug    string
	Content string    `admin:"list label='Page content'"`
	Added   time.Time `admin:"list label='Publish date' format='02.01.2006'"`
}
```


`NameTransform` is a function that takes a string and returns a string. It's used to transform struct field names to database table names. For example, Beego ORM uses snake case versions of struct fields for table / column names, so it'll convert "CompanyEmployee" to "company_employee". This is optional, so if no `NameTransform` is specified, lookups in the database will use the CamelCase versions like in Go.

A struct / model can contain additional information about its fields in a tag. `admin:"-"` means that this field shouldn't show up when editing data. "list" will make this field show up in the table list for this model. `label` allows you to set a custom, human friendly name for a column / field. Multi word labels must be in single quotes. `time.Time` fields also take an optional `format` string.

Screenshots
-----------

![List view](/../master/screenshots/list.png?raw=true "List view")
![New blog post](/../master/screenshots/new.png?raw=true "New blog post")