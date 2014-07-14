package fields

import (
	"html/template"
	"io"
	"time"
)

var timeTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="date" value="{{.value}}" class="form-control" placeholder="{{.format}}">
`))

type TimeField struct {
	*BaseField
	Format string
}

func (t *TimeField) Configure(tagMap map[string]string) error {
	t.Format = "2006-01-02 15:04"
	if format, ok := tagMap["format"]; ok {
		t.Format = format
	}
	return nil
}

func (t *TimeField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	formatted := ""
	if tm, ok := val.(time.Time); ok {
		formatted = tm.Format(t.Format)
	}
	t.BaseRender(w, timeTemplate, formatted, err, startRow, map[string]interface{}{
		"format": t.Format,
	})
}

func (t *TimeField) RenderString(val interface{}) template.HTML {
	if maybeTime, ok := val.(time.Time); ok {
		return template.HTML(template.HTMLEscapeString(maybeTime.Format(t.Format)))
	}
	return template.HTML("")
}

func (t *TimeField) Validate(val string) (interface{}, error) {
	tm, err := time.Parse(t.Format, val)
	if err != nil {
		return nil, err
	}
	return tm, nil
}
