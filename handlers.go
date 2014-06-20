package admin

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type route struct {
	name    string
	method  string
	path    []string
	handler httprouter.Handle
}

type urlConfig struct {
	prefix string
	router *httprouter.Router
	routes map[string]*route
}

func newURLConfig(prefix string) *urlConfig {
	conf := &urlConfig{
		prefix,
		httprouter.New(),
		map[string]*route{},
	}

	return conf
}

func (u *urlConfig) add(name, method string, path string, handler httprouter.Handle) error {
	if _, ok := u.routes[name]; ok {
		return errors.New(fmt.Sprintf("Route \"%v\" already exists.", name))
	}

	pathParts := make([]string, 0, strings.Count(path, ":"))

	inArg := false
	endArg := 0
	startArg := 0
	for i, char := range path {
		if char == ':' {
			inArg = true
			startArg = i
		} else if inArg && char == '/' {
			inArg = false
			pathParts = append(pathParts, path[endArg:startArg])
			endArg = i
		}
		if i == len(path)-1 {
			pathParts = append(pathParts, path[endArg:len(path)])
		}
	}
	u.routes[name] = &route{name, method, pathParts, handler}
	u.router.Handle(method, u.prefix+path, handler)

	return nil
}

func (u *urlConfig) URL(name string, args ...interface{}) (string, error) {
	route, ok := u.routes[name]
	if !ok {
		return "", errors.New("No such route.")
	}

	if len(args) != len(route.path)-1 {
		return "", errors.New(fmt.Sprintf("Needed %v argument(s) for \"%v\" but got %v.", len(route.path)-1, name, len(args)))
	}

	var buf bytes.Buffer
	buf.WriteString(u.prefix)
	for i, part := range route.path {
		buf.WriteString(part)
		if len(args) > i {
			fmt.Fprint(&buf, args[i])
		}
	}
	return buf.String(), nil
}

var templates *template.Template

func (a *Admin) render(rw http.ResponseWriter, req *http.Request, tmpl string, ctx map[string]interface{}) {
	ctx["title"] = a.title
	ctx["path"] = a.Path
	ctx["q"] = req.Form.Get("q")
	if _, ok := ctx["anonymous"]; !ok {
		ctx["anonymous"] = false
	}

	sess := a.getUserSession(req)
	if sess != nil {
		ctx["messages"] = sess.getMessages()
	}

	err := templates.ExecuteTemplate(rw, tmpl, ctx)
	if err != nil {
		fmt.Println(err)
	}
}

// handlerWrapper is used to redirect to index / log in page.
func (a *Admin) handlerWrapper(h httprouter.Handle) httprouter.Handle {
	return func(rw http.ResponseWriter, req *http.Request, params httprouter.Params) {
		if a.getUserSession(req) == nil && req.URL.Path != a.Path+"/" {
			http.Redirect(rw, req, a.Path, 302)
			return
		}
		h(rw, req, params)
	}
}

func (a *Admin) handleIndex(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	if a.getUserSession(req) == nil {
		if req.Method == "POST" {
			req.ParseForm()
			ok := a.logIn(rw, req.Form.Get("username"), req.Form.Get("password"))
			if ok {
				http.Redirect(rw, req, a.Path, 302)
			}
		}
		a.render(rw, req, "login.html", map[string]interface{}{
			"anonymous": true,
		})
		return
	}
	a.render(rw, req, "index.html", map[string]interface{}{
		"groups": a.modelGroups,
	})
}

func (a *Admin) handleLogout(rw http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	cookie, err := req.Cookie("admin")
	if err != nil {
		return
	}

	delete(a.sessions, cookie.Value)
	http.Redirect(rw, req, a.Path, 302)
}
func (a *Admin) handleList(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	slug := ps.ByName("slug")

	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	// Columns
	columns := []string{}
	colNames := []string{}
	for _, field := range model.fields {
		if field.Attrs().List {
			columns = append(columns, field.Attrs().Label)
			colNames = append(colNames, field.Attrs().Name)
		}
	}

	// GET parameters
	req.ParseForm()
	q := req.Form.Get("q")

	// Sort
	sortBy := req.Form.Get("sort")
	if len(sortBy) == 0 {
		sortBy = model.sort
	}
	sortDesc := false
	if sortBy[0] == '-' {
		sortBy = sortBy[1:]
		sortDesc = true
	}

	if model.fieldByName(sortBy) == nil {
		sortBy = ""
	}

	// Page number
	page, err := strconv.ParseUint(req.Form.Get("page"), 10, 64)
	if err != nil {
		page = 1
	}

	// Get data
	results, rows, err := model.page(int(page), q, sortBy, sortDesc)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Invalid page
	if len(results) == 0 && page != 1 {
		sess := a.getUserSession(req)
		sess.addMessage("warning", "Empty page.")
		http.Redirect(rw, req, req.URL.Path, 302)
		return
	}

	// Render / format field data
	strResults := [][]template.HTML{}
	fields := model.listFields
	for _, row := range results {
		s := make([]template.HTML, len(row))
		for i, val := range row {
			s[i] = fields[i].RenderString(val)
		}
		strResults = append(strResults, s)
	}

	// Full list view or popup window
	var tmpl string
	if view := ps.ByName("view"); view == "popup" {
		tmpl = "popup.html"
	} else {
		tmpl = "list.html"
	}

	// Page numbers
	pages := make([]int, int(float64(rows)/25.0+0.5))
	for i, _ := range pages {
		pages[i] = i + 1
	}

	a.render(rw, req, tmpl, map[string]interface{}{
		"name": model.Name,
		"slug": slug,

		"columns":  columns,
		"colNames": colNames,
		"sort":     sortBy,
		"sortDesc": sortDesc,

		"results": strResults,

		"page":     page,
		"numPages": len(pages),
		"pages":    pages,
		"rows":     rows,
	})
}

func (a *Admin) handleEdit(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var data map[string]interface{}
	var errors map[string]string
	if req.Method == "POST" {
		data, errors = a.handleSave(rw, req, ps)
		if data == nil {
			return
		}
	}

	// The model we're editing
	slug := ps.ByName("slug")
	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	// Get ID if we're editing something
	var id int
	if idStr := ps.ByName("id"); len(idStr) > 0 {
		id64, err := strconv.ParseInt(idStr, 10, 64)
		id = int(id64)
		if err != nil {
			http.NotFound(rw, req)
			return
		}
	}

	// If no errors / not yet submitted for validation, and we're editing, get data from db
	if errors == nil && id != 0 {
		var err error
		data, err = model.get(id)
		if err != nil {
			http.NotFound(rw, req)
			return
		}
	}

	// Render form and template
	var buf bytes.Buffer
	model.renderForm(&buf, data, id == 0, errors)

	a.render(rw, req, "edit.html", map[string]interface{}{
		"id":   id,
		"name": model.Name,
		"slug": model.Slug,
		"form": template.HTML(buf.String()),
	})
}

func (a *Admin) handleSave(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) (map[string]interface{}, map[string]string) {
	err := req.ParseMultipartForm(1024 * 1000)
	if err != nil {
		return nil, nil
	}

	slug := ps.ByName("slug")
	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return nil, nil
	}

	id := 0
	if idStr := ps.ByName("id"); len(idStr) > 0 {
		id, err = parseInt(idStr)
		if err != nil {
			return nil, nil
		}
	}

	sess := a.getUserSession(req)
	data, dataErrors, err := model.save(id, req)
	if err != nil {
		sess.addMessage("warning", err.Error())

		// Error && data == nil means no changes were made
		if data == nil {
			url, _ := a.urls.URL("edit", slug, id)
			http.Redirect(rw, req, url, 302)
		}
		return data, dataErrors
	} else {
		sess.addMessage("success", fmt.Sprintf("%v has been saved.", model.Name))
		if req.Form.Get("done") == "true" {
			url, _ := a.urls.URL("view", slug)
			http.Redirect(rw, req, url, 302)
		} else {
			url, _ := a.urls.URL("edit", slug, id)
			http.Redirect(rw, req, url, 302)
		}
		return nil, nil
	}
}

func (a *Admin) handleDelete(rw http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	slug := ps.ByName("slug")
	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	id := 0
	if idStr := ps.ByName("id"); len(idStr) > 0 {
		var err error
		id, err = parseInt(idStr)
		if err != nil {
			return
		}
	}

	err := model.delete(id)
	sess := a.getUserSession(req)
	if err == nil {
		sess.addMessage("success", fmt.Sprintf("%v has been deleted.", model.Name))
	} else {
		sess.addMessage("warning", err.Error())
	}

	http.Redirect(rw, req, a.modelURL(slug, "view", 0), 302)
	return
}
