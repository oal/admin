package fields

import (
	"fmt"
	"html/template"
	"io"
	"net/url"
)

var urlTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="url" value="{{.value}}" class="form-control" placeholder="http://">
	{{if .help}}
		<div class="help text">
			<pre>{{.help}}</pre>
		</div>
	{{end}}
`))

type URLField struct {
	*BaseField
}

func (n *URLField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	n.BaseRender(w, urlTemplate, val, err, startRow, nil)
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
