package admin

import (
	"errors"
	"html/template"
	"io"
	"time"
)

var fieldTemplates, _ = template.ParseGlob(
	"admin/templates/fields/*.html",
)

type Widget interface {
	Configure(map[string]string) error
	Render(io.Writer, string, interface{})
	Validate(string) (interface{}, error)
}

// Text widget

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

// Number widget

type NumberWidget struct {
}

func (n *NumberWidget) Configure(tagMap map[string]string) error {
	return nil
}

func (n *NumberWidget) Render(w io.Writer, name string, val interface{}) {
	fieldTemplates.ExecuteTemplate(w, "Number.html", map[string]interface{}{
		"name":  name,
		"value": val,
	})
}
func (n *NumberWidget) Validate(val string) (interface{}, error) {
	num, err := parseInt(val)
	if err != nil {
		return nil, err
	}
	return num, nil
}

// URL widget

type URLWidget struct {
}

func (n *URLWidget) Configure(tagMap map[string]string) error {
	return nil
}

func (n *URLWidget) Render(w io.Writer, name string, val interface{}) {
	fieldTemplates.ExecuteTemplate(w, "URL.html", map[string]interface{}{
		"name":  name,
		"value": val,
	})
}
func (n *URLWidget) Validate(val string) (interface{}, error) {
	return val, nil
}

// Time widget

type TimeWidget struct {
	Format string
}

func (n *TimeWidget) Configure(tagMap map[string]string) error {
	n.Format = "02.01.2006"
	return nil
}

func (n *TimeWidget) Render(w io.Writer, name string, val interface{}) {
	formatted := ""
	if t, ok := val.(time.Time); ok {
		formatted = t.Format(n.Format)
	}
	fieldTemplates.ExecuteTemplate(w, "Time.html", map[string]interface{}{
		"name":   name,
		"format": n.Format,
		"value":  formatted,
	})
}
func (n *TimeWidget) Validate(val string) (interface{}, error) {
	t, err := time.Parse(n.Format, val)
	if err != nil {
		return nil, err
	}
	return t, nil
}
