package main

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oal/admin"
	"net/http"
	"strings"
	"time"
)

type Category struct {
	Id          int    `orm:"auto"`
	Title       string `admin:"list search"`
	Description string `admin:"list blank null default='No description.'" orm:"null"`
}

func (c *Category) SortBy() string {
	return "Title"
}

type BlogPost struct {
	Id        int       `orm:"auto"`
	Category  *Category `admin:"list='Title' label='Category' width=2" orm:"rel(fk)"` // list='Title' is used to show Category.Title instead of Category.Id in list view.
	Title     string    `admin:"list search width=7"`
	Photo     string    `admin:"width=3 field='file' upload_to='static/posts'"` // File field
	Body      string    `admin:"textarea" orm:"type(text)"`
	Published time.Time `admin:"list width=11"`
	Draft     bool      `admin:"list width=1"`
}

func (b *BlogPost) AdminName() string {
	return "Blog post"
}

func main() {
	// Beego related
	orm.RegisterDataBase("default", "sqlite3", "db.sqlite")
	orm.RegisterModel(new(Category))
	orm.RegisterModel(new(BlogPost))
	orm.RunCommand()

	// Admin related

	// Set up atmin
	a, err := admin.Setup(&admin.Admin{
		Path:     "/admin", // Where you want to access admin. Absolute path, without trailing slash
		Username: "admin",
		Password: "example",
	})
	if err != nil {
		panic(err)
	}
	a.SetTitle("Example admin")
	a.SetDatabase("sqlite3", "db.sqlite")
	a.SetNameTransformer(snakeString) // Optional, but needed here to be compatible with Beego ORM

	group, err := a.Group("Blog")
	if err != nil {
		panic(err)
	}
	group.RegisterModel(new(Category))
	group.RegisterModel(new(BlogPost))

	adminHandler, err := a.Handler()
	if err != nil {
		panic(err)
	}

	router := http.NewServeMux()
	router.Handle("/admin/", adminHandler)
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Nothing to see here. Visit /admin/ instead."))
	})

	http.ListenAndServe(":8000", router)
}

// snakeString converts struct fields from CamelCase to snake_case for Beego ORM
func snakeString(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	return strings.ToLower(string(data[:len(data)]))
}
