package fig

import (
	"reflect"
	"testing"
)

func Test_flattenCfg(t *testing.T) {
	type J struct {
		K bool
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
		}
		G *struct {
			H int
		}
		i int
		J
	}{}
	cfg.B.C = []struct{ D *int }{{}, {}}
	cfg.E = &struct{ F []string }{}

	fields := flattenCfg(&cfg)
	if len(fields) != 10 {
		t.Fatalf("len(fields) == %d, expected %d", len(fields), 10)
	}
	checkField(t, fields[0], "A", "A")
	checkField(t, fields[1], "B", "B")
	checkField(t, fields[2], "C", "B.C")
	checkField(t, fields[3], "D", "B.C[0].D")
	checkField(t, fields[4], "D", "B.C[1].D")
	checkField(t, fields[5], "E", "E")
	checkField(t, fields[6], "F", "E.F")
	checkField(t, fields[7], "G", "G")
	checkField(t, fields[8], "J", "J")
	checkField(t, fields[9], "K", "J.K")
}

func Test_flattenField(t *testing.T) {
	t.Run("struct field", func(t *testing.T) {
		cfg := struct {
			A int `fig:"a,required"`
		}{}

		f := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}
		fields := make([]*field, 0)
		flattenField(f, &fields)

		if len(fields) != 1 {
			t.Fatalf("len(fields) == %d, expected %d", len(fields), 1)
		}
		if fields[0].sliceIdx != -1 {
			t.Errorf("sliceIdx == %d, expected %d", fields[0].sliceIdx, -1)
		}
		if fields[0].parent != f {
			t.Errorf("parent == %p, expected %p", fields[0].parent, f)
		}
		if fields[0].v.Kind() != reflect.Int {
			t.Errorf("kind == %v, expected %v", fields[0].v.Kind(), reflect.Int)
		}
		if !fields[0].v.CanSet() {
			t.Errorf("f.CanSet() == false")
		}
		if fields[0].v.Type() != reflect.TypeOf(cfg.A) {
			t.Errorf("type == %v, expected %v", fields[0].v.Kind(), reflect.TypeOf(cfg.A))
		}
		if fields[0].st.Tag.Get("fig") != "a,required" {
			t.Errorf("tag == %s, expected %s", fields[0].st.Tag.Get("fig"), "a,required")
		}
	})

	t.Run("slice field", func(t *testing.T) {
		cfg := struct {
			A []struct {
				B int `fig:"b"`
			} `fig:"a"`
		}{}
		cfg.A = []struct {
			B int `fig:"b"`
		}{{B: 5}}

		f := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}
		fields := make([]*field, 0)
		flattenField(f, &fields)

		if len(fields) != 2 {
			t.Fatalf("len(fields) == %d, expected %d", len(fields), 1)
		}
		if fields[1].sliceIdx != -1 {
			t.Errorf("sliceIdx == %d, expected %d", fields[1].sliceIdx, -1)
		}
		if fields[1].parent.sliceIdx != 0 {
			t.Errorf("parent.sliceIdx == %d, expected %d", fields[1].parent.sliceIdx, 0)
		}
		if fields[1].parent.parent != fields[0] {
			t.Errorf("parent.parent == %p, expected %p", fields[1].parent.parent, fields[0])
		}
		if fields[1].v.Kind() != reflect.Int {
			t.Errorf("kind == %v, expected %v", fields[1].v.Kind(), reflect.Int)
		}
		if !fields[1].v.CanSet() {
			t.Errorf("f.CanSet() == false")
		}
		if fields[1].st.Tag.Get("fig") != "b" {
			t.Errorf("tag == %s, expected %s", fields[1].st.Tag.Get("fig"), "b")
		}
	})
}

func Test_parseTagVal(t *testing.T) {
	for _, tc := range []struct {
		tagVal string
		want   structTag
		err    bool
	}{
		{
			tagVal: "",
			want:   structTag{},
		},
		{
			tagVal: "a",
			want:   structTag{name: "a"},
		},
		{
			tagVal: "a,",
			want:   structTag{name: "a"},
		},
		{
			tagVal: "a,default=go",
			want:   structTag{name: "a", defaultVal: "go"},
		},
		{
			tagVal: "b,required",
			want:   structTag{name: "b", required: true},
		},
		{
			tagVal: "b,default=d,required",
			err:    true,
		},
		{
			tagVal: "b,mandatory",
			err:    true,
		},
	} {
		t.Run(tc.tagVal, func(t *testing.T) {
			tag, err := parseTagVal(tc.tagVal)
			if tc.err {
				if err == nil {
					t.Fatalf("parseTagVal() returned nil error")
				}
			} else {
				if err != nil {
					t.Fatalf("parseTagVal() returned unexpected error: %v", err)
				}
				if !reflect.DeepEqual(tc.want, tag) {
					t.Fatalf("parseTagVal() == %+v, expected %+v", tag, tc.want)
				}
			}
		})
	}

}

func Test_field_parseTag(t *testing.T) {
	cfg := struct {
		A string `fig:"a,default=go"`
	}{}

	f := &field{
		st: reflect.ValueOf(&cfg).Elem().Type().Field(0),
	}
	want := structTag{name: "a", defaultVal: "go"}
	tag, err := f.parseTag("fig")
	if err != nil {
		t.Fatalf("f.parseTag() returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(want, tag) {
		t.Fatalf("f.parseTag() == %+v, expected %+v", tag, want)
	}
	// check that tag is set inside field as well
	if !reflect.DeepEqual(want, f.tag) {
		t.Fatalf("f.tag == %+v, expected %+v", tag, want)
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

func Test_splitTag(t *testing.T) {
	for _, tc := range []struct {
		S    string
		Want []string
	}{
		{
			S:    ",[hello, world]",
			Want: []string{"", "[hello, world]"},
		},
		{
			S:    ",required",
			Want: []string{"", "required"},
		},
		{
			S:    "single",
			Want: []string{"single"},
		},
		{
			S:    "log,required,[1,2,3]",
			Want: []string{"log", "required", "[1,2,3]"},
		},
		{
			S:    "log,[55.5,8.2],required",
			Want: []string{"log", "[55.5,8.2]", "required"},
		},
		{
			S:    "語,[語,foo本bar],required",
			Want: []string{"語", "[語,foo本bar]", "required"},
		},
	} {
		t.Run(tc.S, func(t *testing.T) {
			got := splitTag(tc.S)

			if len(tc.Want) != len(got) {
				t.Fatalf("want len %d, got %d", len(tc.Want), len(got))
			}

			for i, val := range tc.Want {
				if got[i] != val {
					t.Errorf("want slice[%d] == %s, got %s", i, val, got[i])
				}
			}
		})
	}
}
