package fields

import (
	"html/template"
	"io"
	"strconv"
	"strings"
)

var m2mTemplate = template.Must(template.New("template").Parse(`
	<div class="input-group">
		<input id="{{.name}}" name="{{.name}}" type="text" value="{{.value}}" class="form-control">
		<span class="input-group-btn">
			<button class="btn btn-default btn-fk-search" type="button" data-name="{{.name}}" data-slug="{{.modelSlug}}">Search...</button>
		</span>
	</div>
`))

type ManyToManyField struct {
	*BaseField

	table  string
	column string
	model  string
}

func (m *ManyToManyField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	m.BaseRender(w, m2mTemplate, val, err, startRow, map[string]interface{}{
		"modelSlug": m.model,
	})
}

func (m *ManyToManyField) Validate(val string) (interface{}, error) {
	idStr := strings.Split(val, ",")
	ids := []int{}

	for _, s := range idStr {
		s = strings.TrimSpace(s)
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}

		ids = append(ids, int(id))
	}

	return ids, nil
}

func (m *ManyToManyField) SetRelatedTable(table string) {
	m.table = table
}

func (m *ManyToManyField) GetRelatedTable() string {
	return m.table
}

func (m *ManyToManyField) SetListColumn(column string) {
	m.column = column
}

func (m *ManyToManyField) GetListColumn() string {
	return m.column
}

func (m *ManyToManyField) SetModelSlug(slug string) {
	m.model = slug
}

func (m *ManyToManyField) GetModelSlug() string {
	return m.model
}
