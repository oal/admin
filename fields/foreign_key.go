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

	table  string
	column string
	model  string
}

func (f *ForeignKeyField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	f.BaseRender(w, foreignKeyTemplate, val, err, startRow, map[string]interface{}{
		"modelSlug": f.model,
	})
}
func (f *ForeignKeyField) Validate(val string) (interface{}, error) {
	return val, nil
}

func (f *ForeignKeyField) SetRelatedTable(table string) {
	f.table = table
}

func (f *ForeignKeyField) GetRelatedTable() string {
	return f.table
}

func (f *ForeignKeyField) SetListColumn(column string) {
	f.column = column
}

func (f *ForeignKeyField) GetListColumn() string {
	return f.column
}

func (f *ForeignKeyField) SetModelSlug(slug string) {
	f.model = slug
}

func (f *ForeignKeyField) GetModelSlug() string {
	return f.model
}

func (f *ForeignKeyField) GetRelationTable() string {
	return ""
}
