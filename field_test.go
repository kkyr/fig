package fig

import "testing"

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
