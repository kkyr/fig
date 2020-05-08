package fig

import (
	"fmt"
	"strings"
)

const (
	requiredKey = "required"
	defaultKey  = "default="
)

// parseFieldTag parses a field's tag using the tag with tagKey
// into a fieldTag. any error during parsing is stored inside
// err and should be checked before accessing other fields.
func parseFieldTag(f *field, tagKey string) (ft fieldTag) {
	vals := splitTag(f.st.Tag.Get(tagKey))
	if len(vals) == 0 {
		return
	}
	ft.name = strings.TrimSpace(vals[0])
	vals = vals[1:]
	if len(vals) == 0 {
		return
	}

	if len(vals) > 1 {
		ft.err = fmt.Errorf("too many keys in tag")
		return
	}

	switch {
	case vals[0] == requiredKey:
		ft.required = true
	case strings.HasPrefix(vals[0], defaultKey):
		ft.defaultVal = vals[0][len(defaultKey):]
		if len(ft.defaultVal) == 0 {
			ft.err = fmt.Errorf("default value is empty")
		}
	default:
		ft.err = fmt.Errorf("unexpected tag key: %s", vals[0])
	}

	return
}

// fieldTag is a parsed tag of a field.
type fieldTag struct {
	err        error  // non-nil if there was any error during parsing
	required   bool   // true if the tag contained a required key
	defaultVal string // default value if tag container a default key
	name       string // the name of the field as described by the tag, if any
}

// splitTag is like strings.Split with a comma separator but it does not
// flattenStruct values inside square brackets (f.sliceStart & f.sliceEnd).
// "ports,default=[80,443]" ---> []string{"port", "default=[80,443]"}
// if you pass a string with brackets within brackets then god help you.
func splitTag(tag string) []string {
	var (
		res      []string
		start    = 0
		inString = false
	)

	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' && !inString { // nolint: gocritic
			res = append(res, tag[start:i])
			start = i + 1
		} else if tag[i] == '[' {
			inString = true
		} else if tag[i] == ']' {
			inString = false
		}
	}

	if start < len(tag) {
		res = append(res, tag[start:])
	}

	return res
}
