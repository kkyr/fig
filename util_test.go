package fig

import (
	"reflect"
	"testing"
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
