package fields

import (
	"errors"
	"html/template"
	"io"
	"strconv"
)

var textTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="text" value="{{.value}}" class="form-control">
`))

var textareaTemplate = template.Must(template.New("template").Parse(`
	<textarea id="{{.name}}" name="{{.name}}" class="form-control">{{.value}}</textarea>
`))

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
		length, err := strconv.ParseInt(maxLength, 10, 64)
		if err != nil {
			return err
		}
		t.MaxLength = int(length)
	}
	return nil
}

func (t *TextField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	tmpl := textTemplate
	if t.isTextarea {
		tmpl = textareaTemplate
	}
	t.BaseRender(w, tmpl, val, err, startRow, nil)
}
func (t *TextField) Validate(val string) (interface{}, error) {
	if t.MaxLength != 0 && len(val) > t.MaxLength {
		return nil, errors.New("Value is too long")
	}
	return val, nil
}
