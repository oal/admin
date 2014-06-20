Admin
=====

Web based admin interface for Go, inspired by Django admin.

### Features

-   Single user login (TODO: Implement support for custom login / user handlers).
-   Register and group structs as "models" that map to your database manually or via an ORM.
-   Set custom attributes via each struct field's tag to choose which columns are shown in lists, searchable etc (see below).
-   Search, list and sort rows.
-   Custom formatting of values like time.Time etc.
-   Override / add custom fields with custom validation, formatting etc (may not work at the moment, but will soon).
-   Auto generate forms from structs for easy content management. Foreign keys and ManyToMany relationships are supported, as long as target struct is also registered (choose by ID or via popup window).

### Example

See the example app in /example for a working test app.

In simple terms, this is how you activate it (likely to change soon):

```go
a, err := admin.Setup(&admin.Admin{
	Path: "/admin",

	Username: "admin",
	Password: "password"
})
if err != nil {
	panic(err)
}

a.SetTitle("My admin panel")
a.SetDatabase("sqlite3", "db.sqlite")
a.SetNameTransformer(snakeString)

g, err := a.Group("Main")
if err != nil {
	panic(err)
}
g.RegisterModel(new(Page))

handler, err := a.Handler() // Attach handler to any mux to serve the admin panel.
if err != nil {
	panic(err)
}
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

### Struct tags

Additional options can be provided in the `admin` struct tag, as in the example above. If more than one is used, separate them by a single space ` `. Multiple word values must be single quoted. Currently, these are supported:

-   `-` Skip / hide column (id / first column can't be hidden)
-   `list` Show column in list view
    -   `list='FieldName'` is available for pointers / `ForeignKeyField`s and will display RelatedField.FieldName instead of its Id value.
-   `search` Make column searchable
-   `blank` Allow this field to be empty.
-   `null` Only works if `blank` is used. Instead of inserting empty values, NULL will be used for empty fields.
-   `field=file` Lets you specify a non-default field type. `url` and `file` are currently supported
    -   `file` also takes an optional `upload_to='some/path'`
-   `label='Custom name'` Custom label for column
-   `default='My default value'` Default value in "new"/"create" form
-   `width=4` Custom field width / column width (Optional, if not specified, 12 / full width is default)
-   `format='01.02.2006'` For time.Time fields
-   `textarea` Used by string / text field to display field as a textarea instead of an input

This project is still early in development. More documentation and features will be added over time.

### Screenshots (outdated)

![List view](/../master/screenshots/list.png?raw=true "List view")
![New blog post](/../master/screenshots/new.png?raw=true "New blog post")