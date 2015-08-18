package fields

import (
	"fmt"
	"html/template"
	"io"
	"strconv"
)

type BooleanField struct {
	*BaseField
}

func (b *BooleanField) Configure(tagMap map[string]string) error {
	b.Blank = true
	return nil
}

func (b *BooleanField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	b.BaseRender(w, val, err, startRow, nil)
}

func (b *BooleanField) Validate(val string) (interface{}, error) {
	bl, err := strconv.ParseBool(val)
	if err != nil {
		return false, nil
	}
	return bl, nil
}

func (b *BooleanField) RenderString(val interface{}) template.HTML {
	s := `<span class="glyphicon %v"></span>`
	if i, ok := val.(int64); ok && i == 1 {
		s = fmt.Sprintf(s, "text-success glyphicon-ok")
	} else {
		s = fmt.Sprintf(s, "text-danger glyphicon-remove")
	}
	return template.HTML(template.HTML(s))
}

func (b *BooleanField) BaseRender(w io.Writer, value interface{}, errStr string, startRow bool, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = map[string]interface{}{}
	}
	ctx["label"] = b.Label
	ctx["blank"] = b.Blank
	ctx["name"] = b.Name
	ctx["value"] = value
	ctx["error"] = errStr
	ctx["help"] = b.Help
	ctx["startrow"] = startRow
	ctx["width"] = 12
	ctx["right"] = b.Right

	err := fieldWrapper_bool.Execute(w, ctx)
	if err != nil {
		fmt.Println(err)
	}

}

var fieldWrapper_bool = template.Must(template.New("FieldWrapper").Parse(`
	{{if .startrow}}</div><div class="row">{{end}}
	<div class="col-sm-{{.width}}">
		<div class="form-group">
			{{if not .right}}<input style="height:30px; width:30px;" id="{{.name}}" name="{{.name}}" type="checkbox" value="true"{{if .value}} checked{{end}}>{{end}}
			<label for="{{.name}}">{{.label}}{{if not .blank}} *{{end}}</label>
			{{if .right}}<input style="height:30px; width:30px;" id="{{.name}}" name="{{.name}}" type="checkbox" value="true"{{if .value}} checked{{end}}>{{end}}
			{{if .help}}
				<div class="help text">
					<pre>{{.help}}</pre>
				</div>
			{{end}}
			{{if .error}}<p class="text-danger">{{.error}}</p>{{end}}
		</div>
	</div>
`))
