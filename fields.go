package admin

import (
	"errors"
	"html/template"
	"io"
)

var fieldTemplates, _ = template.ParseGlob(
	"admin/templates/fields/*.html",
)

type Field interface {
	Render(io.Writer, interface{})
	Validate(interface{}) (interface{}, error)
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
func (b TextField) Validate(val interface{}) (interface{}, error) {
	return val, nil
}

type Widget interface {
	Configure(map[string]string) error
	Render(io.Writer, string, interface{})
	Validate(string) (interface{}, error)
}

type TextWidget struct {
	isTextarea bool
	MaxLength  int
}

func (t *TextWidget) Configure(tagMap map[string]string) error {
	if widget, ok := tagMap["widget"]; ok {
		t.isTextarea = widget == "textarea"
	}
	if maxLength, ok := tagMap["maxlength"]; ok {
		length, err := parseInt(maxLength)
		if err != nil {
			return err
		}
		t.MaxLength = length
	}
	return nil
}

func (t *TextWidget) Render(w io.Writer, name string, val interface{}) {
	tmpl := "TextField.html"
	if t.isTextarea {
		tmpl = "Textarea.html"
	}
	fieldTemplates.ExecuteTemplate(w, tmpl, map[string]interface{}{
		"name":  name,
		"value": val,
	})
}
func (t *TextWidget) Validate(val string) (interface{}, error) {
	if t.MaxLength != 0 && len(val) > t.MaxLength {
		return nil, errors.New("Value is too long")
	}
	return val, nil
}
