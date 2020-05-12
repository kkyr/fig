package fig

import (
	"reflect"
	"testing"
)

func Test_flattenCfg(t *testing.T) {
	type J struct {
		K bool `fig:"k"`
	}
	cfg := struct {
		A string
		B struct {
			C []struct {
				D *int
			}
		}
		E *struct {
			F []string
		} `fig:"e"`
		G *struct {
			H int
		}
		i int
		J
	}{}
	cfg.B.C = []struct{ D *int }{{}, {}}
	cfg.E = &struct{ F []string }{}

	fields := flattenCfg(&cfg, "fig")
	if len(fields) != 10 {
		t.Fatalf("len(fields) == %d, expected %d", len(fields), 10)
	}
	checkField(t, fields[0], "A", "A")
	checkField(t, fields[1], "B", "B")
	checkField(t, fields[2], "C", "B.C")
	checkField(t, fields[3], "D", "B.C[0].D")
	checkField(t, fields[4], "D", "B.C[1].D")
	checkField(t, fields[5], "e", "e")
	checkField(t, fields[6], "F", "e.F")
	checkField(t, fields[7], "G", "G")
	checkField(t, fields[8], "J", "J")
	checkField(t, fields[9], "k", "J.k")
}

func Test_newStructField(t *testing.T) {
	cfg := struct {
		A int `fig:"a" default:"5" validate:"required"`
	}{}
	parent := &field{
		v:        reflect.ValueOf(&cfg).Elem(),
		t:        reflect.ValueOf(&cfg).Elem().Type(),
		sliceIdx: -1,
	}

	f := newStructField(parent, 0, "fig")
	if f.parent != parent {
		t.Errorf("f.parent == %p, expected %p", f.parent, f)
	}
	if f.sliceIdx != -1 {
		t.Errorf("f.sliceIdx == %d, expected %d", f.sliceIdx, -1)
	}
	if f.v.Kind() != reflect.Int {
		t.Errorf("f.v.Kind == %v, expected %v", f.v.Kind(), reflect.Int)
	}
	if f.v.Type() != reflect.TypeOf(cfg.A) {
		t.Errorf("f.v.Type == %v, expected %v", f.v.Kind(), reflect.TypeOf(cfg.A))
	}
	if f.altName != "a" {
		t.Errorf("f.altName == %s, expected %s", f.altName, "a")
	}
	if !f.required {
		t.Errorf("f.required == false")
	}
	if !f.setDefault {
		t.Errorf("f.setDefault == false")
	}
	if f.defaultVal != "5" {
		t.Errorf("f.defaultVal == %s, expected %s", f.defaultVal, "5")
	}
}

func Test_newSliceField(t *testing.T) {
	cfg := struct {
		A []struct {
			B int
		} `fig:"aaa"`
	}{}
	cfg.A = []struct {
		B int
	}{{B: 5}}

	parent := &field{
		v:        reflect.ValueOf(&cfg).Elem().Field(0),
		t:        reflect.ValueOf(&cfg).Elem().Field(0).Type(),
		st:       reflect.ValueOf(&cfg).Elem().Type().Field(0),
		sliceIdx: -1,
	}

	f := newSliceField(parent, 0, "fig")
	if f.parent != parent {
		t.Errorf("f.parent == %p, expected %p", f.parent, f)
	}
	if f.sliceIdx != 0 {
		t.Errorf("f.sliceIdx == %d, expected %d", f.sliceIdx, 0)
	}
	if f.v.Kind() != reflect.Struct {
		t.Errorf("f.v.Kind == %v, expected %v", f.v.Kind(), reflect.Int)
	}
	if f.altName != "aaa" {
		t.Errorf("f.altName == %s, expected %s", f.altName, "a")
	}
	if f.required {
		t.Errorf("f.required == true")
	}
	if f.setDefault {
		t.Errorf("f.setDefault == true")
	}
	if f.defaultVal != "" {
		t.Errorf("f.defaultVal == %s, expected %s", f.defaultVal, "")
	}
}

func Test_parseTag(t *testing.T) {
	for _, tc := range []struct {
		tagVal string
		want   structTag
	}{
		{
			tagVal: "",
			want:   structTag{},
		},
		{
			tagVal: `fig:"a"`,
			want:   structTag{altName: "a"},
		},
		{
			tagVal: `fig:"a,"`,
			want:   structTag{altName: "a"},
		},
		{
			tagVal: `fig:"a" default:"go"`,
			want:   structTag{altName: "a", setDefault: true, defaultVal: "go"},
		},
		{
			tagVal: `fig:"b" validate:"required"`,
			want:   structTag{altName: "b", required: true},
		},
		{
			tagVal: `fig:"b" validate:"required" default:"go"`,
			want:   structTag{altName: "b", required: true, setDefault: true, defaultVal: "go"},
		},
		{
			tagVal: `fig:"c,omitempty"`,
			want:   structTag{altName: "c"},
		},
	} {
		t.Run(tc.tagVal, func(t *testing.T) {
			tag := parseTag(reflect.StructTag(tc.tagVal), "fig")
			if !reflect.DeepEqual(tc.want, tag) {
				t.Fatalf("parseTag() == %+v, expected %+v", tag, tc.want)
			}
		})
	}
}

func checkField(t *testing.T, f *field, name, path string) {
	t.Helper()
	if f.name() != name {
		t.Errorf("f.name() == %s, expected %s", f.name(), name)
	}
	if f.path() != path {
		t.Errorf("f.path() == %s, expected %s", f.path(), path)
	}
}
