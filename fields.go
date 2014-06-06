package admin

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"time"
)

type Field interface {
	Configure(map[string]string) error
	Render(w io.Writer, val interface{}, err string, startRow bool)
	RenderString(val interface{}) template.HTML
	Validate(string) (interface{}, error)
	Attrs() *BaseField
}

var customFields = map[string]Field{
	"url": &URLField{&BaseField{}},
}

type BaseField struct {
	name         string
	label        string
	defaultValue interface{}
	columnName   string
	list         bool
	searchable   bool
	width        int
}

func (b *BaseField) Configure(tagMap map[string]string) error {
	return nil
}

func (b *BaseField) Validate(val string) (interface{}, error) {
	return val, nil
}

func (b *BaseField) RenderString(val interface{}) template.HTML {
	return template.HTML(template.HTMLEscapeString(fmt.Sprintf("%v", val)))
}

func (b *BaseField) Attrs() *BaseField {
	return b
}

func (b *BaseField) BaseRender(w io.Writer, tmpl string, value interface{}, errStr string, startRow bool, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = map[string]interface{}{}
	}
	ctx["label"] = b.label
	ctx["name"] = b.name
	ctx["value"] = value
	ctx["error"] = errStr
	ctx["tmpl"] = tmpl

	ctx["startrow"] = startRow
	if b.width == 0 {
		b.width = 12
	}
	ctx["width"] = b.width

	err := templates.ExecuteTemplate(w, "FieldWrapper", ctx)
	if err != nil {
		fmt.Println(err)
	}
}

// Text Field
type TextField struct {
	*BaseField
	isTextarea bool
	MaxLength  int
}

func (t *TextField) Configure(tagMap map[string]string) error {
	if _, ok := tagMap["textarea"]; ok {
		t.isTextarea = true
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

func (t *TextField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	tmpl := "TextField.html"
	if t.isTextarea {
		tmpl = "Textarea.html"
	}
	t.BaseRender(w, tmpl, val, err, startRow, nil)
}
func (t *TextField) Validate(val string) (interface{}, error) {
	if t.MaxLength != 0 && len(val) > t.MaxLength {
		return nil, errors.New("Value is too long")
	}
	return val, nil
}

// Foreign key field
type ForeignKeyField struct {
	*BaseField
	model *model
}

func (t *ForeignKeyField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	t.BaseRender(w, "ForeignKey.html", val, err, startRow, map[string]interface{}{
		"modelSlug": t.model.Slug,
	})
}
func (t *ForeignKeyField) Validate(val string) (interface{}, error) {
	return val, nil
}

// Int field
type IntField struct {
	*BaseField
	step int
	min  *int
	max  *int
}

func (i *IntField) Configure(tagMap map[string]string) error {
	step := 1
	if str, ok := tagMap["step"]; ok {
		var err error
		step, err = parseInt(str)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		min, err := parseInt(str)
		if err != nil {
			return err
		}
		i.min = &min
	}
	if str, ok := tagMap["max"]; ok {
		max, err := parseInt(str)
		if err != nil {
			return err
		}
		i.max = &max
	}
	i.step = step
	return nil
}

func (i *IntField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	i.BaseRender(w, "Number.html", val, err, startRow, map[string]interface{}{
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

// Float field
type FloatField struct {
	*BaseField
	step float64
	min  *float64
	max  *float64
}

func (f *FloatField) Configure(tagMap map[string]string) error {
	step := 0.01
	if str, ok := tagMap["step"]; ok {
		var err error
		step, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		min, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		f.min = &min
	}
	if str, ok := tagMap["max"]; ok {
		max, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		f.max = &max
	}
	f.step = step
	return nil
}

func (f *FloatField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	f.BaseRender(w, "Number.html", val, err, startRow, map[string]interface{}{
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

// URL field
type URLField struct {
	*BaseField
}

func (n *URLField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	n.BaseRender(w, "URL.html", val, err, startRow, nil)
}

func (n *URLField) RenderString(val interface{}) template.HTML {
	return template.HTML(fmt.Sprintf("<a href=\"%v\">%v</a>", val, val))
}

func (n *URLField) Validate(val string) (interface{}, error) {
	_, err := url.Parse(val)
	if err != nil {
		return nil, err
	}
	return val, nil
}

// Time field
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

func (n *TimeField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	formatted := ""
	if t, ok := val.(time.Time); ok {
		formatted = t.Format(n.Format)
	}
	n.BaseRender(w, "Time.html", formatted, err, startRow, map[string]interface{}{
		"format": n.Format,
	})
}

func (n *TimeField) RenderString(val interface{}) template.HTML {
	return template.HTML(template.HTMLEscapeString(val.(time.Time).Format(n.Format)))
}

func (n *TimeField) Validate(val string) (interface{}, error) {
	t, err := time.Parse(n.Format, val)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Boolean field
type BooleanField struct {
	*BaseField
}

func (b *BooleanField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	b.BaseRender(w, "Boolean.html", val, err, startRow, nil)
}

func (b *BooleanField) Validate(val string) (interface{}, error) {
	bl, err := strconv.ParseBool(val)
	if err != nil {
		return false, nil
	}
	return bl, nil
}
