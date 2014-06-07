Admin
=====

Web based admin interface for Go, inspired by Django admin.

It currently only works with SQLite, but Postgres and MySQL support should be easy to implement. See the example app in /example for a working test app.

The code is a bit rough, but feel free to submit pull requests and / or open issues in the issue tracker to discuss or report bugs, feature requests etc.

If you want to try it in your own app, you can activate like this:

```go
router := mux.NewRouter()
a, err := admin.Setup(&admin.Admin{
	Title:         "My admin panel",
	Router:        router,
	Path:          "/admin",
	Database:      "db.sqlite",
	NameTransform: snakeString,

	UploadDir: "/path/to/uploads",

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
	Id      int
	Name    string    `admin:"list search width=6"` // Listed, searchable and half width (12 columns is full width)
	Slug    string    `admin:"list search width=6"` // Listed, searchable and half width (displayed to the right of Name in edit form)
	Content string    `admin:"list textarea label='Page content'"`
	Added   time.Time `admin:"list label='Publish date' format='02.01.2006'"`
}
```

`NameTransform` is a function that takes a string and returns a string. It's used to transform struct field names to database table names. For example, Beego ORM uses snake case versions of struct fields for table / column names, so it'll convert "CompanyEmployee" to "company_employee". This is optional, so if no `NameTransform` is specified, lookups in the database will use the CamelCase versions like in Go.

Struct tag
----------

Additional options can be provided in the `admin` struct tag, as in the example above. If more than one is used, separate them by a single space ` `. Multiple word values must be single quoted. Currently, these are supported:

-   `-` Skip / hide column (id / first column can't be hidden)
-   `list` Show column in list view
-   `search` Make column searchable
-   `field=file` Lets you specify a non-default field type. `url` and `file` are currently supported
    -   `file` also takes an optional `upload_to='some/path'`
-   `label='Custom name'` Custom label for column
-   `default='My default value'` Default value in "new"/"create" form
-   `width=4` Custom field width / column width (Optional, if not specified, 12 / full width is default)
-   `format='01.02.2006'` For time.Time fields
-   `textarea` Used by string / text field to display field as a textarea instead of an input

This project is still early in development. More documentation and features will be added over time.

Screenshots
-----------

![List view](/../master/screenshots/list.png?raw=true "List view")
![New blog post](/../master/screenshots/new.png?raw=true "New blog post")