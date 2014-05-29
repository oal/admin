package admin

import (
	"html/template"
	"io"
)

var fieldTemplates, _ = template.ParseGlob(
	"admin/templates/fields/*.html",
)

type Field interface {
	Render(io.Writer, interface{})
	Validate(interface{}) bool
}

type TextField struct {
	Name string
}

func (b TextField) Render(w io.Writer, val interface{}) {
	fieldTemplates.ExecuteTemplate(w, "TextField.html", map[string]interface{}{
		"name":  b.Name,
		"value": val,
	})
}
func (b TextField) Validate(val interface{}) bool {
	return true
}
