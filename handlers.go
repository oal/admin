package admin

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/oal/admin/fields"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

var templates *template.Template

func (a *Admin) render(rw http.ResponseWriter, req *http.Request, tmpl string, ctx map[string]interface{}) {
	ctx["title"] = a.Title
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
func (a *Admin) handlerWrapper(h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if a.getUserSession(req) == nil && req.URL.Path != a.Path+"/" {
			http.Redirect(rw, req, a.Path, 302)
			return
		}
		h.ServeHTTP(rw, req)
	}
}

func (a *Admin) handleIndex(rw http.ResponseWriter, req *http.Request) {
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

func (a *Admin) handleLogout(rw http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie("admin")
	if err != nil {
		return
	}

	delete(a.sessions, cookie.Value)
	http.Redirect(rw, req, a.Path, 302)
}
func (a *Admin) handleList(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slug := vars["slug"]

	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	columns := []string{}
	for _, field := range model.fields {
		if field.Attrs().List {
			columns = append(columns, field.Attrs().Name)
		}
	}

	req.ParseForm()
	q := req.Form.Get("q")

	results, err := a.queryModel(model, q)
	if err != nil {
		fmt.Println(err)
		return
	}

	strResults := [][]template.HTML{}
	fields := model.listFields
	for _, row := range results {
		s := make([]template.HTML, len(row))
		for i, val := range row {
			s[i] = fields[i].RenderString(val)
		}
		strResults = append(strResults, s)
	}

	var tmpl string
	if view, ok := vars["view"]; ok && view == "popup" {
		tmpl = "popup.html"
	} else {
		tmpl = "list.html"
	}

	a.render(rw, req, tmpl, map[string]interface{}{
		"name":    model.Name,
		"slug":    slug,
		"columns": columns,
		"results": strResults,
	})
}

func (a *Admin) handleEdit(rw http.ResponseWriter, req *http.Request) {
	var data map[string]interface{}
	var errors map[string]string
	if req.Method == "POST" {
		data, errors = a.handleSave(rw, req)
		if data == nil {
			return
		}
	}

	// The model we're editing
	vars := mux.Vars(req)
	slug := vars["slug"]

	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	// Get ID if we're editing something
	var id int
	if idStr, ok := vars["id"]; ok {
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
		data, err = a.querySingleModel(model, id)
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

func (a *Admin) handleSave(rw http.ResponseWriter, req *http.Request) (map[string]interface{}, map[string]string) {
	err := req.ParseMultipartForm(1024 * 1000)
	if err != nil {
		return nil, nil
	}

	vars := mux.Vars(req)
	slug := vars["slug"]
	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return nil, nil
	}

	id := 0
	if idStr, ok := vars["id"]; ok {
		id, err = parseInt(idStr)
		if err != nil {
			return nil, nil
		}
	}

	numFields := len(model.fieldNames) - 1 // No need for ID.

	// Get data from POST and fill a slice
	data := map[string]interface{}{}
	errors := map[string]string{}
	hasErrors := false
	for i := 0; i < numFields; i++ {
		fieldName := model.fieldNames[i+1]
		field := model.fieldByName(fieldName)
		rawValue := req.Form.Get(fieldName)

		// If file field (and no rawValue), handle file
		isFile := false
		if fileField, ok := field.(fields.FileHandlerField); ok {
			isFile = true
			files, ok := req.MultipartForm.File[fieldName]
			if ok {
				filename, err := fileField.HandleFile(files[0])
				if err != nil {
					panic(err)
				}
				rawValue = filename
			}
		}

		val, err := field.Validate(rawValue)
		if err != nil {
			errors[fieldName] = err.Error()
			hasErrors = true
		}

		// Files have no regular form value set, so we don't want to set it's data to "" and have it removed
		if rawValue == "" && isFile {
			continue
		}

		data[fieldName] = val
	}

	if hasErrors {
		return data, errors
	}

	// Get existing data, if any, so we can check what values were changed (existing == nil for new rows)
	var existing map[string]interface{}
	if id != 0 {
		existing, err = a.querySingleModel(model, id)
		if err != nil {
			panic(err)
		}
	}

	// Create query only with the changed data
	changedCols := []string{}
	changedData := []interface{}{}
	for key, value := range data {
		// Skip if not changed
		if existing != nil && value == existing[key] {
			continue
		}

		// Convert to DB version of name and append
		col := key
		if a.NameTransform != nil {
			col = a.NameTransform(key)
		}
		if id != 0 {
			col = fmt.Sprintf("%v = ?", col)
		}
		changedCols = append(changedCols, col)
		changedData = append(changedData, value)
	}

	sess := a.getUserSession(req)
	if len(changedCols) == 0 {
		sess.addMessage("warning", fmt.Sprintf("%v was not saved because there were no changes.", model.Name))
		http.Redirect(rw, req, a.modelURL(slug, fmt.Sprintf("/edit/%v", id)), 302)
		return nil, nil
	}

	valMarks := strings.Repeat("?, ", len(data))
	valMarks = valMarks[0 : len(valMarks)-2]

	// Insert / update
	var q string
	if id != 0 {
		q = fmt.Sprintf("UPDATE %v SET %v WHERE id = %v", model.tableName, strings.Join(changedCols, ", "), id)
	} else {
		q = fmt.Sprintf("INSERT INTO %v(%v) VALUES(%v)", model.tableName, strings.Join(changedCols, ", "), valMarks)
	}

	_, err = a.db.Exec(q, changedData...)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	sess.addMessage("success", fmt.Sprintf("%v has been saved.", model.Name))

	if req.Form.Get("done") == "true" {
		http.Redirect(rw, req, a.modelURL(slug, ""), 302)
	} else {
		http.Redirect(rw, req, a.modelURL(slug, fmt.Sprintf("/edit/%v", id)), 302)
	}
	return nil, nil
}

func (a *Admin) handleDelete(rw http.ResponseWriter, req *http.Request) {

	// The model we're editing
	vars := mux.Vars(req)
	slug := vars["slug"]

	model, ok := a.models[slug]
	if !ok {
		http.NotFound(rw, req)
		return
	}

	id := 0
	if idStr, ok := vars["id"]; ok {
		var err error
		id, err = parseInt(idStr)
		if err != nil {
			return
		}
	}

	_, err := a.querySingleModel(model, id)
	if err != nil {
		http.NotFound(rw, req)
		return
	}

	q := fmt.Sprintf("DELETE FROM %v WHERE id=?", model.tableName)
	_, err = a.db.Exec(q, id)
	if err != nil {
		fmt.Println(err)
		return
	}

	sess := a.getUserSession(req)
	sess.addMessage("success", fmt.Sprintf("%v has been deleted.", model.Name))
	http.Redirect(rw, req, a.modelURL(slug, ""), 302)
	return
}
