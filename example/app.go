package main

import (
	"github.com/astaxie/beego/orm"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oal/admin"
	"net/http"
	"strings"
	"time"
)

type Category struct {
	Id          int    `orm:"auto" admin:"-"`
	Title       string `admin:"list search"`
	Description string `admin:"list"`
}

type BlogPost struct {
	Id        int       `orm:"auto" admin:"-"`
	Category  *Category `orm:"rel(fk)" admin:"list width=3"`
	Title     string    `admin:"list search width=9"`
	Body      string    `orm:"type(text)" admin:"textarea"`
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
	router := mux.NewRouter()
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Nothing to see here. Visit /admin/ instead."))
	})

	// Set up atmin
	a, err := admin.Setup(&admin.Admin{
		Title:         "Example admin", // Optional. Without it the admin will simply be called "Admin"
		Router:        router,          // Must be a gorilla mux.Router
		Path:          "/admin",        // Where you want to access admin, without trailing slash
		Database:      "db.sqlite",     // Only SQLite is supported at the moment
		NameTransform: snakeString,     // Optional, but needed here to be compatible with Beego ORM
		Username:      "admin",
		Password:      "example",
	})
	if err != nil {
		panic(err)
	}
	group, err := a.Group("Blog")
	if err != nil {
		panic(err)
	}
	group.RegisterModel(new(Category))
	group.RegisterModel(new(BlogPost))

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
