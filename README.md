Admin
=====

Web based admin interface for Go, inspired by Django admin.

Currently only works with SQLite, no authentication etc. WIP.


Symlink / copy templates (they should be accessible from your-project/admin/templates)

	ln -s $GOPATH/src/github.com/oal/admin admin


Activate like this:

	router := mux.NewRouter()
	a, err := admin.Setup(&admin.Admin{
		Title:         "My admin panel",
		Router:        router,
		Path:          "/admin",
		Database:      "db.sqlite",
		NameTransform: snakeString,
	})
	if err != nil {
		panic(err)
	}

	g, err := a.Group("Main")
	if err != nil {
		panic(err)
	}
	g.RegisterModel(new(Page))


A model is just a struct

	type Page struct {
		Id      int    `admin:"-"`
		Name    string `admin:"list"`
		Slug    string
		Content string `admin:"list"`
	}


`NameTransform` is a function that takes a string and returns a string. It's used to transform struct field names to database table names. For example, Beego ORM uses snake case versions of struct fields for table / column names, so it'll convert "CompanyEmployee" to "company_employee". This is optional, so if no `NameTransform` is specified, lookups in the database will use the CamelCase versions like in Go.

A struct / model can contain additional information about its fields in a tag. `admin:"-"` means that this field shouldn't show up when editing data. "list" will make this field show up in the table list for this model.