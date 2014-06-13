package fields

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
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

var CustomFields = map[string]Field{
	"url":  &URLField{&BaseField{}},
	"file": &FileField{&BaseField{}, ""},
}

type BaseField struct {
	Name         string
	Label        string
	DefaultValue interface{}
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

var fieldWrapper = template.Must(template.New("FieldWrapper").Parse(`
	{{if .startrow}}</div><div class="row">{{end}}
	<div class="col-sm-{{.width}}">
		<div class="form-group">
			<label for="{{.name}}">{{.label}}</label>
			{{.field}}
			{{if .error}}<p class="text-danger">{{.error}}</p>{{end}}
		</div>
	</div>
`))

var numberTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="number" step="{{.step}}"{{if .min}} min="{{.min}}"{{end}}{{if .step}}  max="{{.max}}"{{end}} value="{{.value}}" class="form-control">
`))
