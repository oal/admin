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
	results, rows, err := a.queryModel(model, q, sortBy, sortDesc, int(page))
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
	if view, ok := vars["view"]; ok && view == "popup" {
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

	// Get existing data, if any, so we can check what values were changed (existing == nil for new rows)
	var existing map[string]interface{}
	if id != 0 {
		existing, err = a.querySingleModel(model, id)
		if err != nil {
			panic(err)
		}
	}

	// Get data from POST and fill a slice
	data := map[string]interface{}{}
	m2mData := map[string][]int{}
	errors := map[string]string{}
	hasErrors := false
	for i := 0; i < numFields; i++ {
		fieldName := model.fieldNames[i+1]
		field := model.fieldByName(fieldName)

		var existingVal interface{}
		if existing != nil {
			existingVal = existing[fieldName]
		}

		val, err := fields.Validate(field, req, existingVal)
		if err != nil {
			errors[fieldName] = err.Error()
			hasErrors = true
		}

		// ManyToManyField
		if ids, ok := val.([]int); ok {
			m2mData[fieldName] = ids
			continue
		}

		data[fieldName] = val
	}

	if hasErrors {
		return data, errors
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

	valMarks := strings.Repeat("?, ", len(changedCols))
	valMarks = valMarks[0 : len(valMarks)-2]

	// Insert / update
	var q string
	if id != 0 {
		q = fmt.Sprintf("UPDATE %v SET %v WHERE id = %v", model.tableName, strings.Join(changedCols, ", "), id)
	} else {
		q = fmt.Sprintf("INSERT INTO %v(%v) VALUES(%v)", model.tableName, strings.Join(changedCols, ", "), valMarks)
	}

	result, err := a.db.Exec(q, changedData...)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	if newId, _ := result.LastInsertId(); id == 0 {
		id = int(newId)
	}

	// Insert / update M2M
	for fieldName, ids := range m2mData {
		field, _ := model.fieldByName(fieldName).(*fields.ManyToManyField)

		m2mTable := fmt.Sprintf("%v_%v", model.tableName, field.ColumnName)
		toColumn := fmt.Sprintf("%v_id", field.GetRelatedTable())
		fromColumn := fmt.Sprintf("%v_id", model.tableName)
		q = fmt.Sprintf("SELECT %v FROM %v WHERE %v = ?", toColumn, m2mTable, fromColumn)

		rows, err := a.db.Query(q, id)
		if err != nil {
			return nil, nil
		}

		removeRels := map[int]bool{}
		for rows.Next() {
			var eId int
			rows.Scan(&eId)
			removeRels[eId] = true
		}

		// Add new, remove from removeRels as we go. Those still left in removeRels will be deleted.
		for _, nId := range ids {
			if _, ok := removeRels[nId]; ok {
				// Already exists,
				delete(removeRels, nId)
			} else {
				// Relation doesn't exist yet, so add it
				q = fmt.Sprintf("INSERT INTO %v (%v, %v) VALUES (?, ?)", m2mTable, fromColumn, toColumn)
				a.db.Exec(q, id, nId)
			}
		}

		// Delete remaining Ids in removeRels as they're no longer related
		for eId, _ := range removeRels {
			q = fmt.Sprintf("DELETE FROM %v WHERE %v = ? AND %v = ?", m2mTable, fromColumn, toColumn)
			a.db.Exec(q, id, eId)
		}
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
