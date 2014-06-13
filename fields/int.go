package fields

import (
	"io"
	"strconv"
)

type IntField struct {
	*BaseField
	step int
	min  *int
	max  *int
}

func (i *IntField) Configure(tagMap map[string]string) error {
	step := 1
	if str, ok := tagMap["step"]; ok {
		var err error
		step64, err := strconv.ParseInt(str, 10, 64)
		step = int(step64)
		if err != nil {
			return err
		}
	}
	if str, ok := tagMap["min"]; ok {
		min64, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		min := int(min64)
		i.min = &min
	}
	if str, ok := tagMap["max"]; ok {
		max64, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		max := int(max64)
		i.max = &max
	}
	i.step = step
	return nil
}

func (i *IntField) Render(w io.Writer, val interface{}, err string, startRow bool) {
	i.BaseRender(w, numberTemplate, val, err, startRow, map[string]interface{}{
		"step": i.step,
	})
}
func (i *IntField) Validate(val string) (interface{}, error) {
	num, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil, err
	}
	return num, nil
}
