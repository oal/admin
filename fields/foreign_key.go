package fields

import (
	"html/template"
	"io"
)

var foreignKeyTemplate = template.Must(template.New("template").Parse(`
	<div class="input-group">
		<input id="{{.name}}" name="{{.name}}" type="text" value="{{.value}}" class="form-control">
		<span class="input-group-btn">
			<button class="btn btn-default btn-fk-search" type="button" data-name="{{.name}}" data-slug="{{.modelSlug}}">Search...</button>
		</span>
	</div>
`))

type ForeignKeyField struct {
	*BaseField
	ModelSlug string
}

func (f *ForeignKeyField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	f.BaseRender(w, foreignKeyTemplate, val, err, startRow, map[string]interface{}{
		"modelSlug": f.ModelSlug,
	})
}
func (f *ForeignKeyField) Validate(val string) (interface{}, error) {
	return val, nil
}
