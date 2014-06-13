package fields

import (
	"io"
	"strconv"
)

type FloatField struct {
	*BaseField
	step float64
	min  *float64
	max  *float64
}

func (f *FloatField) Configure(tagMap map[string]string) error {
	step := 0.01
	if str, ok := tagMap["step"]; ok {
		var err error
		step, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		min, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		f.min = &min
	}
	if str, ok := tagMap["max"]; ok {
		max, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		f.max = &max
	}
	f.step = step
	return nil
}

func (f *FloatField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	f.BaseRender(w, numberTemplate, val, err, startRow, map[string]interface{}{
		"step": f.step,
		"min":  f.min,
		"max":  f.max,
	})
}
func (f *FloatField) Validate(val string) (interface{}, error) {
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, err
	}
	return num, nil
}
