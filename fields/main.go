package fields

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
)

type Field interface {
	Configure(map[string]string) error
	Render(w io.Writer, val interface{}, err string, startRow bool)
	RenderString(val interface{}) template.HTML
	Validate(string) (interface{}, error)
	Attrs() *BaseField
}

type FileHandlerField interface {
	HandleFile(*multipart.FileHeader) (string, error)
}

type BaseField struct {
	Name         string
	Label        string
	DefaultValue interface{}
	Optional     bool
	ColumnName   string
	List         bool
	Searchable   bool
	Width        int
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

func (b *BaseField) BaseRender(w io.Writer, tmpl *template.Template, value interface{}, errStr string, startRow bool, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = map[string]interface{}{}
	}
	ctx["label"] = b.Label
	ctx["optional"] = b.Optional
	ctx["name"] = b.Name
	ctx["value"] = value
	ctx["error"] = errStr
	ctx["startrow"] = startRow
	if b.Width == 0 {
		b.Width = 12
	}
	ctx["width"] = b.Width

	var buf bytes.Buffer
	tmpl.Execute(&buf, ctx)
	ctx["field"] = template.HTML(buf.String())

	err := fieldWrapper.Execute(w, ctx)
	if err != nil {
		fmt.Println(err)
	}
}

var customFields = map[string]Field{
	"url":  &URLField{&BaseField{}},
	"file": &FileField{&BaseField{}, ""},
}

func RegisterCustom(name string, field Field) error {
	if _, ok := customFields[name]; ok {
		return errors.New(fmt.Sprintf("A field with the name %v already exists.", name))
	}

	if field.Attrs() == nil {
		return errors.New("Add a *BaseField and other initial values if needed before registering.")
	}

	customFields[name] = field
	return nil
}

func GetCustom(name string) Field {
	if field, ok := customFields[name]; ok {
		return field
	}

	return nil
}

func Validate(field Field, req *http.Request, existing interface{}) (interface{}, error) {
	fieldName := field.Attrs().Name
	rawValue := req.Form.Get(fieldName)

	// If file field (and no rawValue), handle file
	if fileField, ok := field.(FileHandlerField); ok {
		files, ok := req.MultipartForm.File[fieldName]
		if ok {
			filename, err := fileField.HandleFile(files[0])
			if err != nil {
				panic(err)
			}
			rawValue = filename
		} else {
			rawValue = existing.(string)
		}
	}

	return field.Validate(rawValue)
}

var fieldWrapper = template.Must(template.New("FieldWrapper").Parse(`
	{{if .startrow}}</div><div class="row">{{end}}
	<div class="col-sm-{{.width}}">
		<div class="form-group">
			<label for="{{.name}}">{{.label}}{{if not .optional}} *{{end}}</label>
			{{.field}}
			{{if .error}}<p class="text-danger">{{.error}}</p>{{end}}
		</div>
	</div>
`))

var numberTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="number" step="{{.step}}"{{if .min}} min="{{.min}}"{{end}}{{if .step}}  max="{{.max}}"{{end}} value="{{.value}}" class="form-control">
`))
