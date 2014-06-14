package fields

import (
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"os"
)

var fileTemplate = template.Must(template.New("template").Parse(`
	<input id="{{.name}}" name="{{.name}}" type="file">
	<p>{{if .value}}Existing: {{.value}}{{end}}</p>
`))

type FileField struct {
	*BaseField
	UploadTo string
}

func (f *FileField) Configure(tagMap map[string]string) error {
	if dir, ok := tagMap["upload_to"]; ok {
		f.UploadTo = dir
	}
	return nil
}

func (f *FileField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	f.BaseRender(w, fileTemplate, val, err, startRow, nil)
}

func (f *FileField) Validate(val string) (interface{}, error) {
	return val, nil
}

func (f *FileField) HandleFile(file *multipart.FileHeader) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", err
	}

	filename := file.Filename
	if len(f.UploadTo) > 0 {
		_, err := os.Stat(f.UploadTo)
		if err != nil {
			os.MkdirAll(f.UploadTo, 0777)
		}
		filename = fmt.Sprintf("%v/%v", f.UploadTo, file.Filename)
	}

	dst, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	io.Copy(dst, reader)
	dst.Close()
	reader.Close()
	return filename, nil
}
