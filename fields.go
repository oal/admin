package admin

import (
	"errors"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"time"
)

var fieldTemplates, _ = template.ParseGlob(
	"admin/templates/fields/*.html",
)

type Field interface {
	Configure(map[string]string) error
	Render(io.Writer, interface{}, string)
	Validate(string) (interface{}, error)
	Attrs() *BaseField
}

type BaseField struct {
	name       string
	label      string
	columnName string
	list       bool
}

func (b *BaseField) Configure(tagMap map[string]string) error {
	return nil
}

func (b *BaseField) Attrs() *BaseField {
	return b
}
func (b *BaseField) BaseRender(w io.Writer, tmpl string, value interface{}, err string, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = map[string]interface{}{}
	}
	ctx["label"] = b.label
	ctx["name"] = b.name
	ctx["value"] = value
	ctx["error"] = err

	fieldTemplates.ExecuteTemplate(w, tmpl, ctx)
}

// Text Field

type TextField struct {
	*BaseField
	isTextarea bool
	MaxLength  int
}

func (t *TextField) Configure(tagMap map[string]string) error {
	if Field, ok := tagMap["Field"]; ok {
		t.isTextarea = Field == "textarea"
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

func (t *TextField) Render(w io.Writer, val interface{}, err string) {
	tmpl := "TextField.html"
	if t.isTextarea {
		tmpl = "Textarea.html"
	}
	t.BaseRender(w, tmpl, val, err, nil)
}
func (t *TextField) Validate(val string) (interface{}, error) {
	if t.MaxLength != 0 && len(val) > t.MaxLength {
		return nil, errors.New("Value is too long")
	}
	return val, nil
}

type IntField struct {
	*BaseField
	step int
	min  int
	max  int
}

func (i *IntField) Configure(tagMap map[string]string) error {
	step := 1
	min := -100000
	max := 100000

	if str, ok := tagMap["step"]; ok {
		var err error
		step, err = parseInt(str)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		var err error
		min, err = parseInt(str)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["max"]; ok {
		var err error
		max, err = parseInt(str)
		if err != nil {
			return err
		}
	}

	i.step = step
	i.min = min
	i.max = max

	return nil
}

func (i *IntField) Render(w io.Writer, val interface{}, err string) {
	i.BaseRender(w, "Number.html", val, err, map[string]interface{}{
		"step": i.step,
	})
}
func (i *IntField) Validate(val string) (interface{}, error) {
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil, err
	}
	return num, nil
}

type FloatField struct {
	*BaseField
	step float64
	min  float64
	max  float64
}

func (f *FloatField) Configure(tagMap map[string]string) error {
	step := 1.0
	min := -100000.0
	max := 100000.0

	if str, ok := tagMap["step"]; ok {
		var err error
		step, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		var err error
		min, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["max"]; ok {
		var err error
		max, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
	}

	f.step = step
	f.max = max
	f.min = min

	return nil
}

func (f *FloatField) Render(w io.Writer, val interface{}, err string) {
	f.BaseRender(w, "Number.html", val, err, map[string]interface{}{
		"step": f.step,
		"min":  f.min,
		"max":  f.max,
	})
}
func (f *FloatField) Validate(val string) (interface{}, error) {
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, err
	}
	return num, nil
}

// URL Field

type URLField struct {
	*BaseField
}

func (n *URLField) Render(w io.Writer, val interface{}, err string) {
	n.BaseRender(w, "URL.html", val, err, nil)
}
func (n *URLField) Validate(val string) (interface{}, error) {
	_, err := url.Parse(val)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// Time Field

type TimeField struct {
	*BaseField
	Format string
}

func (n *TimeField) Configure(tagMap map[string]string) error {
	n.Format = "2006-02-01"
	if format, ok := tagMap["format"]; ok {
		n.Format = format
	}
	return nil
}

func (n *TimeField) Render(w io.Writer, val interface{}, err string) {
	formatted := ""
	if t, ok := val.(time.Time); ok {
		formatted = t.Format(n.Format)
	}
	n.BaseRender(w, "Time.html", formatted, err, map[string]interface{}{
		"format": n.Format,
	})
}
func (n *TimeField) Validate(val string) (interface{}, error) {
	t, err := time.Parse(n.Format, val)
	if err != nil {
		return nil, err
	}
	return t, nil
}
