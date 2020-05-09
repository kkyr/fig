package fig

import (
	"fmt"
	"reflect"
	"strings"
)

// flattenCfg recursively flattens a cfg struct into
// a slice of its constituent fields.
func flattenCfg(cfg interface{}) []*field {
	root := &field{
		v:        reflect.ValueOf(cfg).Elem(),
		t:        reflect.ValueOf(cfg).Elem().Type(),
		sliceIdx: -1,
	}
	fs := make([]*field, 0)
	flattenField(root, &fs)
	return fs
}

// flattenField recursively flattens a field into its
// constituent fields, filling fs as it goes.
func flattenField(f *field, fs *[]*field) {
	for (f.v.Kind() == reflect.Ptr || f.v.Kind() == reflect.Interface) && !f.v.IsNil() {
		f.v = f.v.Elem()
		f.t = f.v.Type()
	}

	switch f.v.Kind() {
	case reflect.Struct:
		for i := 0; i < f.t.NumField(); i++ {
			unexported := f.t.Field(i).PkgPath != ""
			embedded := f.t.Field(i).Anonymous
			if unexported && !embedded {
				continue
			}
			child := &field{
				parent:   f,
				v:        f.v.Field(i),
				t:        f.v.Field(i).Type(),
				st:       f.t.Field(i),
				sliceIdx: -1,
			}
			*fs = append(*fs, child)
			flattenField(child, fs)
		}

	case reflect.Slice, reflect.Array:
		switch f.t.Elem().Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Ptr, reflect.Interface:
			for i := 0; i < f.v.Len(); i++ {
				child := &field{
					parent:   f,
					v:        f.v.Index(i),
					t:        f.v.Index(i).Type(),
					st:       f.st,
					sliceIdx: i,
				}
				flattenField(child, fs)
			}
		}
	}
}

// field is a settable field of a config object.
type field struct {
	parent *field

	v        reflect.Value
	t        reflect.Type
	st       reflect.StructField
	sliceIdx int // >=0 if this field is a member of a slice.

	tag structTag // populated during parseTag.
}

// name is the name of the field. if the field's struct tag has
// been parsed and contains a name then that name is used, else
// it fallbacks to the field's name as defined in the struct.
// if this field is a member of a slice, then its name is simply
// its index in the slice.
func (f *field) name() string {
	if f.sliceIdx >= 0 {
		return fmt.Sprintf("[%d]", f.sliceIdx)
	}
	if f.tag.name != "" {
		return f.tag.name
	}
	return f.st.Name
}

// path is a dot separated path consisting of the names of the
// field's topmost parent all the way down to the field itself.
func (f *field) path() (path string) {
	var visit func(f *field)
	visit = func(f *field) {
		if f.parent != nil {
			visit(f.parent)
		}
		path += f.name()
		// if it's a slice/array we don't want a dot before the slice indexer
		// e.g. we want A[0].B instead of A.[0].B
		if f.t.Kind() != reflect.Slice && f.t.Kind() != reflect.Array {
			path += "."
		}
	}
	visit(f)
	return strings.Trim(path, ".")
}

// parseTag parses a field's struct tag under tagName and populates
// f's tag field with the result.
func (f *field) parseTag(tagName string) error {
	const (
		requiredKey = "required"
		defaultKey  = "default="
	)

	vals := splitTag(f.st.Tag.Get(tagName))
	switch len(vals) {
	case 0:
		return nil
	case 1:
		f.tag.name = strings.TrimSpace(vals[0])
	case 2:
		f.tag.name = strings.TrimSpace(vals[0])
		f.tag.required = vals[1] == requiredKey
		if strings.HasPrefix(vals[1], defaultKey) {
			f.tag.defaultVal = vals[1][len(defaultKey):]
		}
		if !f.tag.required && len(f.tag.defaultVal) == 0 {
			return fmt.Errorf("invalid tag value: %s", vals[1])
		}
	default:
		return fmt.Errorf("too many values (%d) in tag", len(vals))
	}

	return nil
}

// structTag contains information gathered after parsing a field's
// struct tag.
type structTag struct {
	name       string // the name of the field as defined in the tag.
	required   bool   // true if the tag contains a required key.
	defaultVal string // default value if tag contains a default key.
}

// splitTag behaves like strings.FieldsFunc with a comma separator
// but it does not split comma separated values that are located inside
// square brackets.
//
// Examples:
//   "ports,default=[80,443] --> []string{"port", "default=[80,443]"}
//   ",required"			 --> []string{"", "required"}
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
