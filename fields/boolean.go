package fields

import (
	"html/template"
	"io"
	"strconv"
)

var booleanTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="checkbox" value="true" class="form-control"{{if .value}} checked{{end}}>
`))

type BooleanField struct {
	*BaseField
}

func (b *BooleanField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	b.BaseRender(w, booleanTemplate, val, err, startRow, nil)
}

func (b *BooleanField) Validate(val string) (interface{}, error) {
	bl, err := strconv.ParseBool(val)
	if err != nil {
		return false, nil
	}
	return bl, nil
}
