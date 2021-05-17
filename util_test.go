package fig

import (
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"
)

func Test_stringSlice(t *testing.T) {
	for _, tc := range []struct {
		In   string
		Want []string
	}{
		{
			In:   "false",
			Want: []string{"false"},
		},
		{
			In:   "1,5,2",
			Want: []string{"1", "5", "2"},
		},
		{
			In:   "[hello , world]",
			Want: []string{"hello ", " world"},
		},
		{
			In:   "[foo]",
			Want: []string{"foo"},
		},
	} {
		t.Run(tc.In, func(t *testing.T) {
			got := stringSlice(tc.In)
			if !reflect.DeepEqual(tc.Want, got) {
				t.Fatalf("want %+v, got %+v", tc.Want, got)
			}
		})
	}
}

func Test_fileExists(t *testing.T) {
	dir := filepath.Join("testdata", "valid")
	ok := fileExists(dir)
	if ok {
		t.Errorf("fileExists(dir) == true, expected false")
	}

	bad := "oompa-loompa"
	ok = fileExists(bad)
	if ok {
		t.Errorf("fileExists(bad) == true, expected false")
	}

	good := "fig.go"
	ok = fileExists(good)
	if !ok {
		t.Errorf("fileExists(good) == false, expected true")
	}
}

func Test_isStructPtr(t *testing.T) {
	type cfgType struct{}

	var cfg cfgType
	ok := isStructPtr(cfg)
	if ok {
		t.Errorf("isStructPtr(cfg) == true, expected false")
	}

	ok = isStructPtr(&cfg)
	if !ok {
		t.Errorf("isStructPtr(*cfg) == false, expected true")
	}

	var i int
	ok = isStructPtr(&i)
	if ok {
		t.Errorf("isStructPtr(*i) == true, expected false")
	}
}

func Test_isZero(t *testing.T) {
	t.Run("nil slice is zero", func(t *testing.T) {
		var s []string
		if isZero(reflect.ValueOf(s)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("empty slice is zero", func(t *testing.T) {
		s := []string{}
		if isZero(reflect.ValueOf(s)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("nil pointer is zero", func(t *testing.T) {
		var s *string
		if isZero(reflect.ValueOf(s)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("non-nil pointer is not zero", func(t *testing.T) {
		var a *string
		b := "b"
		a = &b

		if isZero(reflect.ValueOf(a)) == true {
			t.Fatalf("isZero == true")
		}
	})

	t.Run("struct is not zero", func(t *testing.T) {
		a := struct {
			B string
		}{}

		if isZero(reflect.ValueOf(a)) == true {
			t.Fatalf("isZero == true")
		}
	})

	t.Run("zero time is zero", func(t *testing.T) {
		td := time.Time{}

		if isZero(reflect.ValueOf(td)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("non-zero time is not zero", func(t *testing.T) {
		td := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

		if isZero(reflect.ValueOf(td)) == true {
			t.Fatalf("isZero == true")
		}
	})

	t.Run("zero regexp is zero", func(t *testing.T) {
		var re *regexp.Regexp

		if isZero(reflect.ValueOf(re)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("non-zero regexp is not zero", func(t *testing.T) {
		re := regexp.MustCompile(".*")

		if isZero(reflect.ValueOf(re)) == true {
			t.Fatalf("isZero == true")
		}
	})

	t.Run("reflect invalid is zero", func(t *testing.T) {
		var x interface{}

		if isZero(reflect.ValueOf(&x).Elem().Elem()) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("0 int is zero", func(t *testing.T) {
		x := 0

		if isZero(reflect.ValueOf(x)) == false {
			t.Fatalf("isZero == false")
		}
	})

	t.Run("5 int is not zero", func(t *testing.T) {
		x := 5

		if isZero(reflect.ValueOf(x)) == true {
			t.Fatalf("isZero == true")
		}
	})
}
