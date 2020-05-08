package fig

import (
	"fmt"
	"reflect"
	"strings"
)

// field is a settable field of a config object.
type field struct {
	parent *field
	v      reflect.Value
	t      reflect.Type
	st     reflect.StructField
	idx    int // if >=0 then this is a field in a slice indexed at idx

	tag fieldTag
}

// name is the name of the field as reported by the tag.
// if the field has no tag name then the name is the name
// of the field in the struct. if this field is an element
// of a slice then its name is the index in the slice.
func (f *field) name() string {
	if f.idx >= 0 {
		return fmt.Sprintf("[%d]", f.idx)
	}
	if f.tag.name != "" {
		return f.tag.name
	}
	return f.st.Name
}

// path is the full path of the field starting from the
// root of the config struct, joining each successive
// field with a dot.
func (f *field) path() string {
	var path string

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

// flattenStruct recursively flattens a cfg struct into
// a slice of its constituent fields.
func (g *fig) flattenStruct(cfg interface{}) []*field {
	root := &field{
		v:   reflect.ValueOf(cfg).Elem(),
		t:   reflect.ValueOf(cfg).Elem().Type(),
		idx: -1,
	}
	fs := make([]*field, 0)
	g.flattenField(root, &fs)
	return fs
}

// flattenField recursively flattens a field into its
// constituent fields, filling fs as it goes.
func (g *fig) flattenField(f *field, fs *[]*field) {
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
				parent: f,
				v:      f.v.Field(i),
				t:      f.v.Field(i).Type(),
				st:     f.t.Field(i),
				idx:    -1,
			}
			child.tag = parseFieldTag(child, g.tag)
			*fs = append(*fs, child)
			g.flattenField(child, fs)
		}

	case reflect.Slice, reflect.Array:
		switch f.t.Elem().Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Ptr, reflect.Interface:
			for i := 0; i < f.v.Len(); i++ {
				child := &field{
					parent: f,
					v:      f.v.Index(i),
					t:      f.v.Index(i).Type(),
					st:     f.st,
					idx:    i,
				}
				child.tag = parseFieldTag(child, g.tag)
				g.flattenField(child, fs)
			}
		}
	}
}
