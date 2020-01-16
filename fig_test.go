package fig

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type Pod struct {
	APIVersion string `fig:"apiVersion,default=v1"`
	Kind       string `fig:"kind,required"`
	Metadata   struct {
		Name           string        `fig:"name"`
		Environments   []string      `fig:"environments,default=[dev,staging,prod]"`
		Master         bool          `fig:"master,required"`
		MaxPercentUtil *float64      `fig:"maxPercentUtil,default=0.5"`
		Retry          time.Duration `fig:"retry,default=10s"`
	} `fig:"metadata"`
	Spec Spec `fig:"spec"`
}

type Spec struct {
	Containers []Container `fig:"containers"`
	Volumes    []*Volume   `fig:"volumes"`
}

type Container struct {
	Name      string   `fig:"name,required"`
	Image     string   `fig:"image,required"`
	Command   []string `fig:"command"`
	Env       []Env    `fig:"env"`
	Ports     []Port   `fig:"ports"`
	Resources struct {
		Limits struct {
			CPU string `fig:"cpu"`
		} `fig:"limits"`
		Requests *struct {
			Memory string  `fig:"memory,default=64Mi"`
			CPU    *string `fig:"cpu,default=250m"`
		}
	} `fig:"resources"`
	VolumeMounts []VolumeMount `fig:"volumeMounts"`
}

type Env struct {
	Name  string `fig:"name"`
	Value string `fig:"value"`
}

type Port struct {
	ContainerPort int `fig:"containerPort,required"`
}

type VolumeMount struct {
	MountPath string `fig:"mountPath,required"`
	Name      string `fig:"name,required"`
}

type Volume struct {
	Name      string     `fig:"name,required"`
	ConfigMap *ConfigMap `fig:"configMap"`
}

type ConfigMap struct {
	Name  string `fig:"name,required"`
	Items []Item `fig:"items,required"`
}

type Item struct {
	Key  string `fig:"key,required"`
	Path string `fig:"path,required"`
}

func validPodConfig() Pod {
	var pod Pod

	pod.APIVersion = "v1"
	pod.Kind = "Pod"
	pod.Metadata.Name = "redis"
	pod.Metadata.Environments = []string{"dev", "staging", "prod"}
	pod.Metadata.Master = true
	pod.Metadata.Retry = 10 * time.Second
	percentUtil := 0.5
	pod.Metadata.MaxPercentUtil = &percentUtil
	pod.Spec.Containers = []Container{
		{
			Name:  "redis",
			Image: "redis:5.0.4",
			Command: []string{
				"redis-server",
				"/redis-master/redis.conf",
			},
			Env: []Env{
				{
					Name:  "MASTER",
					Value: "true",
				},
			},
			Ports: []Port{
				{ContainerPort: 6379},
			},
			VolumeMounts: []VolumeMount{
				{
					MountPath: "/redis-master-data",
					Name:      "data",
				},
				{
					MountPath: "/redis-master",
					Name:      "config",
				},
			},
		},
	}
	pod.Spec.Containers[0].Resources.Limits.CPU = "0.1"
	pod.Spec.Volumes = []*Volume{
		{Name: "data"},
		{
			Name: "config",
			ConfigMap: &ConfigMap{
				Name: "example-redis-config",
				Items: []Item{
					{
						Key:  "redis-config",
						Path: "redis.conf",
					},
				},
			},
		},
	}

	return pod
}

func Test_fieldErrors(t *testing.T) {
	fe := make(fieldErrors)

	fe["B"] = fmt.Errorf("berr")
	fe["A"] = fmt.Errorf("aerr")

	got := fe.Error()

	want := "A: aerr, B: berr"
	if want != got {
		t.Fatalf("want %q, got %q", want, got)
	}

	fe = make(fieldErrors)
	got = fe.Error()

	if got != "" {
		t.Fatalf("empty errors returned non-empty string: %s", got)
	}
}

func Test_fig_Load(t *testing.T) {
	for _, f := range []string{"pod.yaml", "pod.json", "pod.toml"} {
		t.Run(f, func(t *testing.T) {
			var cfg Pod
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			want := validPodConfig()

			if !reflect.DeepEqual(want, cfg) {
				t.Errorf("\nwant %+v\ngot %+v", want, cfg)
			}
		})
	}
}

func Test_fig_Load_Required(t *testing.T) {
	for _, f := range []string{"pod.yaml", "pod.json", "pod.toml"} {
		t.Run(f, func(t *testing.T) {
			var cfg Pod
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "invalid")))
			if err == nil {
				t.Fatalf("expected err")
			}

			want := []string{
				"Kind",
				"Metadata.Master",
				"Spec.Containers[0].Image",
				"Spec.Volumes[0].ConfigMap.Items",
				"Spec.Volumes[1].Name",
			}

			fieldErrs := err.(fieldErrors)

			if len(want) != len(fieldErrs) {
				t.Fatalf("\nwant len(fieldErrs) == %d, got %d\nerrs: %+v\n", len(want), len(fieldErrs), fieldErrs)
			}

			for _, field := range want {
				if _, ok := fieldErrs[field]; !ok {
					t.Errorf("want %s in fieldErrs, got %+v", field, fieldErrs)
				}
			}
		})
	}
}

func Test_fig_Load_Defaults(t *testing.T) {
	t.Run("non-zero values are not overridden", func(t *testing.T) {
		for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
			t.Run(f, func(t *testing.T) {
				type Server struct {
					Host   string `fig:"host,default=127.0.0.1"`
					Ports  []int  `fig:"ports,default=[80,443]"`
					Logger struct {
						LogLevel   string `fig:"log_level,default=info"`
						Production bool   `fig:"production,default=true"`
						Metadata   struct {
							Keys []string `fig:"keys,default=[ts]"`
						}
					}
					Application struct {
						BuildDate time.Time `fig:"build_date,default=2020-01-01T12:00:00Z"`
					}
				}

				var want Server
				want.Host = "0.0.0.0"
				want.Ports = []int{80, 443}
				want.Logger.LogLevel = "debug"
				want.Logger.Production = true
				want.Logger.Metadata.Keys = []string{"ts"}
				want.Application.BuildDate = time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)

				var cfg Server
				err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}

				if !reflect.DeepEqual(want, cfg) {
					t.Errorf("\nwant %+v\ngot %+v", want, cfg)
				}
			})
		}
	})

	t.Run("bad defaults reported as errors", func(t *testing.T) {
		for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
			t.Run(f, func(t *testing.T) {
				type Server struct {
					Host   string `fig:"host,default=127.0.0.1"`
					Ports  []int  `fig:"ports,default=[80,not-a-port]"`
					Logger struct {
						LogLevel   string `fig:"log_level,default=info"`
						Production bool   `fig:"production,default=not-a-bool"`
						Metadata   struct {
							Keys []string `fig:"keys,required"`
						}
					}
					Application struct {
						BuildDate time.Time `fig:"build_date,default=not-a-time"`
					}
				}

				var cfg Server
				err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
				if err == nil {
					t.Fatalf("expected err")
				}

				want := []string{
					"Ports",
					"Logger.Production",
					"Logger.Metadata.Keys",
					"Application.BuildDate",
				}

				fieldErrs := err.(fieldErrors)

				if len(want) != len(fieldErrs) {
					t.Fatalf("\nlen(fieldErrs) != %d\ngot %+v\n", len(want), fieldErrs)
				}

				for _, field := range want {
					if _, ok := fieldErrs[field]; !ok {
						t.Errorf("want %s in fieldErrs, got %+v", field, fieldErrs)
					}
				}
			})
		}
	})
}

func Test_fig_Load_RequiredAndDefaults(t *testing.T) {
	for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
		t.Run(f, func(t *testing.T) {
			type Server struct {
				Host   string `fig:"host,default=127.0.0.1"`
				Ports  []int  `fig:"ports,required"`
				Logger struct {
					LogLevel   string `fig:"log_level,required"`
					Production bool   `fig:"production,default=5"`
					Metadata   struct {
						Keys []string `fig:"keys,required"`
					}
				}
				Application struct {
					BuildDate time.Time `fig:"build_date,default=2020-01-01T12:00:00Z"`
				}
			}

			var cfg Server
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
			if err == nil {
				t.Fatalf("expected err")
			}

			want := []string{
				"Ports",
				"Logger.Production",
				"Logger.Metadata.Keys",
			}

			fieldErrs := err.(fieldErrors)

			if len(want) != len(fieldErrs) {
				t.Fatalf("\nlen(fieldErrs) != %d\ngot %+v\n", len(want), fieldErrs)
			}

			for _, field := range want {
				if _, ok := fieldErrs[field]; !ok {
					t.Errorf("want %s in fieldErrs, got %+v", field, fieldErrs)
				}
			}
		})
	}
}

func Test_fig_Load_Options(t *testing.T) {
	for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
		t.Run(f, func(t *testing.T) {
			type Server struct {
				Host   string `custom:"host,default=127.0.0.1"`
				Ports  []int  `custom:"ports,default=[80,443]"`
				Logger struct {
					LogLevel   string `custom:"log_level,default=info"`
					Production bool   `custom:"production,default=true"`
					Metadata   struct {
						Keys []string `custom:"keys,default=[ts]"`
					}
				}
				Application struct {
					BuildDate time.Time `custom:"build_date,default=12-25-2012"`
				}
			}

			var want Server
			want.Host = "0.0.0.0"
			want.Ports = []int{80, 443}
			want.Logger.LogLevel = "debug"
			want.Logger.Production = true
			want.Logger.Metadata.Keys = []string{"ts"}
			want.Application.BuildDate = time.Date(2012, 12, 25, 0, 0, 0, 0, time.UTC)

			var cfg Server

			err := Load(&cfg,
				File(f),
				Dirs(filepath.Join("testdata", "valid")),
				Tag("custom"),
				TimeLayout("01-02-2006"),
			)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if !reflect.DeepEqual(want, cfg) {
				t.Errorf("\nwant %+v\ngot %+v", want, cfg)
			}
		})
	}
}

func Test_fig_findFile(t *testing.T) {
	t.Run("finds existing file", func(t *testing.T) {
		fig := newDefaultFig()
		fig.filename = "pod.yaml"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findFile()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		want := filepath.Join("testdata", "valid", "pod.yaml")
		if want != file {
			t.Fatalf("want file %s, got %s", want, file)
		}
	})

	t.Run("non-existing file returns error", func(t *testing.T) {
		fig := newDefaultFig()
		fig.filename = "nope.nope"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findFile()
		if err == nil {
			t.Fatalf("expected err, got file %s", file)
		}
	})
}

func Test_fig_decodeMap(t *testing.T) {
	fig := newDefaultFig()
	fig.tag = "fig"

	m := map[string]interface{}{
		"log_level": "debug",
		"severity":  "5",
		"server": map[string]interface{}{
			"ports":  []int{443, 80},
			"secure": 1,
		},
	}

	var cfg struct {
		Level    string `fig:"log_level"`
		Severity int    `fig:"severity,required"`
		Server   struct {
			Ports  []string `fig:"ports,default=[443]"`
			Secure bool
		} `fig:"server"`
	}

	err := fig.decodeMap(m, &cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if cfg.Level != "debug" {
		t.Errorf("cfg.Level: want %s, got %s", "debug", cfg.Level)
	}

	if cfg.Severity != 5 {
		t.Errorf("cfg.Severity: want %d, got %d", 5, cfg.Severity)
	}

	if reflect.DeepEqual([]int{443, 80}, cfg.Server.Ports) {
		t.Errorf("cfg.Server.Ports: want %+v, got %+v", []int{443, 80}, cfg.Server.Ports)
	}

	if cfg.Server.Secure == false {
		t.Error("cfg.Server.Secure == false")
	}
}

func Test_fig_validate(t *testing.T) {
	fig := newDefaultFig()
	fig.tag = "fig"

	t.Run("struct without tags does nothing", func(t *testing.T) {
		var cfg struct {
			A string
			B int
		}

		cfg.A = "foo"
		cfg.B = 9

		err := fig.validate(&cfg)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if cfg.A != "foo" {
			t.Fatalf("cfg.A: want %s, got %s", "foo", cfg.A)
		}

		if cfg.B != 9 {
			t.Fatalf("cfg.B: want %d, got %d", 9, cfg.B)
		}
	})

	t.Run("non pointer struct returns error", func(t *testing.T) {
		var cfg struct {
			A string `fig:"a,default=foo"`
		}

		err := fig.validate(cfg)
		if err == nil {
			t.Fatal("expected err")
		}
	})

	t.Run("returns field errors as error", func(t *testing.T) {
		var cfg struct {
			A string  `fig:"a,required"`
			D float32 `fig:"d,default=true"`
		}

		err := fig.validate(&cfg)
		if err == nil {
			t.Fatal("expected err")
		}

		fieldErrs, ok := err.(fieldErrors)
		if !ok {
			t.Fatalf("want err type %T, got %T; val %v", fieldErrors{}, err, err)
		}

		if len(fieldErrs) != 2 {
			t.Fatalf("want %d err, got %d; val %+v", 2, len(fieldErrs), fieldErrs)
		}
	})
}

func Test_fig_validateStruct(t *testing.T) {
	fig := newDefaultFig()
	fig.tag = "fig"

	t.Run("struct nil ptr ptr validated", func(t *testing.T) {
		type A struct {
			B string  `fig:",required"`
			X float32 `fig:",default=not-a-float"`
		}

		C := struct {
			A ***A `fig:",required"`
		}{}

		errs := make(fieldErrors)
		fig.validateStruct(reflect.ValueOf(&C).Elem(), errs, "")

		if len(errs) != 1 {
			t.Fatalf("want len(errs) == 1, got %d\nerrs == %+v", len(errs), errs)
		}

		if _, ok := errs["A"]; !ok {
			t.Fatalf("want A in errs, got %v", errs)
		}
	})

	t.Run("struct non-nil ptr inner fields validated", func(t *testing.T) {
		type A struct {
			B string  `fig:",required"`
			X float32 `fig:",default=not-a-float"`
		}

		C := struct {
			A **A
		}{}

		a := &A{}
		C.A = &a

		errs := make(fieldErrors)
		fig.validateStruct(reflect.ValueOf(&C).Elem(), errs, "")

		if len(errs) != 2 {
			t.Fatalf("want len(errs) == 2, got %d\nerrs == %+v", len(errs), errs)
		}

		if _, ok := errs["A.B"]; !ok {
			t.Fatalf("want A.B in errs, got %v", errs)
		}

		if _, ok := errs["A.X"]; !ok {
			t.Fatalf("want A.X in errs, got %v", errs)
		}
	})

	t.Run("nested structs validated", func(t *testing.T) {
		var test struct {
			A string `fig:",required"`
			B struct {
				C int `fig:",default=5"`
				D struct {
					E *float32 `fig:",default=0.125"`
				}
			}
		}

		test.A = "foo"

		var (
			fv   = reflect.ValueOf(&test).Elem()
			errs = make(fieldErrors)
			name = "test"
		)

		fig.validateStruct(fv, errs, name)
		if len(errs) > 0 {
			t.Fatalf("unexpected err: %v", errs)
		}

		if test.A != "foo" {
			t.Fatalf("test.A: want %s, got %s", "foo", test.A)
		}

		if test.B.C != 5 {
			t.Fatalf("test.B.C: want %d, got %d", 5, test.B.C)
		}

		if *test.B.D.E != 0.125 {
			t.Fatalf("test.B.D.E: want %fig, got %fig", 0.125, *test.B.D.E)
		}
	})

	t.Run("slice field names set", func(t *testing.T) {
		type A struct {
			B string `fig:",required"`
		}

		type C struct {
			As []A `fig:"required"`
		}

		type I struct {
			X int `fig:",required"`
		}

		D := struct {
			*I
			Cs []C `fig:"required"`
		}{}

		D.I = &I{}
		D.Cs = []C{
			{
				As: []A{{}, {}},
			},
		}

		errs := make(fieldErrors)
		fig.validateStruct(reflect.ValueOf(D), errs, "")

		if len(errs) == 0 {
			t.Fatalf("expected err")
		}

		if len(errs) != 3 {
			t.Fatalf("expected len(errs) == 3, got %d\nerrs = %+v", len(errs), errs)
		}

		wants := []string{"I.X", "Cs[0].As[0].B", "Cs[0].As[1].B"}
		for _, want := range wants {
			if _, ok := errs[want]; !ok {
				t.Errorf("want %s in errs, got %+v", want, errs)
			}
		}
	})

	t.Run("returns all field errors", func(t *testing.T) {
		var test struct {
			A string `fig:",required"`
			B struct {
				C int `fig:",badkey"`
				D *struct {
					E *float32 `fig:",required"`
				}
				I interface{} `fig:",default=5"`
				S string      `fig:",required"`
			}
		}

		test.B.S = "ok"

		var (
			fv   = reflect.ValueOf(&test).Elem()
			errs = make(fieldErrors)
			name = "test"
		)

		fig.validateStruct(fv, errs, name)
		if len(errs) == 0 {
			t.Fatal("expected err")
		}

		// test.B.D.E not reported as an error as *D is nil
		wantErrs := []string{"test.A", "test.B.C", "test.B.I"}
		if len(wantErrs) != len(errs) {
			t.Fatalf("want %d errs, got %d", len(wantErrs), len(errs))
		}

		for _, want := range wantErrs {
			if _, ok := errs[want]; !ok {
				t.Fatalf("want %s in errs, instead contains %+v", want, errs)
			}
		}
	})
}

func Test_fig_validateField(t *testing.T) {
	fig := newDefaultFig()
	fig.tag = "fig"

	t.Run("nil struct does not validate inner fields", func(t *testing.T) {
		A := struct {
			B *struct {
				C string `fig:"C,required"`
				D bool   `fig:"D,required"`
			}
		}{}

		fv := reflect.ValueOf(A).Field(0)
		fd := reflect.ValueOf(A).Type().Field(0)

		errs := make(fieldErrors)
		fig.validateField(fv, fd, errs, "")

		if len(errs) > 0 {
			t.Fatalf("unexpected err: %v", errs)
		}
	})

	t.Run("struct wrapped in interface validated", func(t *testing.T) {
		A := struct {
			I interface{}
		}{}

		C := struct {
			D string `fig:",required"`
			E int    `fig:",default=5"`
		}{}

		A.I = &C

		fv := reflect.ValueOf(A).Field(0)
		fd := reflect.ValueOf(A).Type().Field(0)

		errs := make(fieldErrors)
		fig.validateField(fv, fd, errs, "")

		if len(errs) != 1 {
			t.Fatalf("want len(errs) == 1, got %d\nerrs = %+v", len(errs), errs)
		}

		if _, ok := errs["I.D"]; !ok {
			t.Fatalf("want I.D in errs, got %+v\n", errs)
		}

		if C.E != 5 {
			t.Fatalf("want C.E == 5, got %d", C.E)
		}
	})
}

func Test_fig_validateCollection(t *testing.T) {
	fig := newDefaultFig()
	fig.tag = "fig"

	t.Run("slice of struct ptrs", func(t *testing.T) {
		type A struct {
			B string `fig:",required"`
			S []int  `fig:",required"`
		}

		C := struct {
			As []*A
		}{
			As: []*A{
				{},
			},
		}

		errs := make(fieldErrors)
		fig.validateCollection(reflect.ValueOf(&C), errs, "")

		if len(errs) == 0 {
			t.Fatalf("expected error")
		}

		if len(errs) != 2 {
			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 2, len(errs), errs)
		}

		for _, want := range []string{"As[0].B", "As[0].S"} {
			if _, ok := errs[want]; !ok {
				t.Fatalf("want %s in errs, got %+v", want, errs)
			}
		}
	})

	t.Run("anonymous struct", func(t *testing.T) {
		type A struct {
			B string `fig:",required"`
			D int    `fig:",default=5"`
		}

		C := struct {
			A
		}{}

		errs := make(fieldErrors)
		fig.validateCollection(reflect.ValueOf(&C).Elem(), errs, "")

		if len(errs) != 1 {
			t.Fatalf("want len(errs) == 1, got %d\nerrs = %+v", len(errs), errs)
		}

		if _, ok := errs["A.B"]; !ok {
			t.Errorf("want A.B in errs, got %+v", errs)
		}

		if C.D != 5 {
			t.Errorf("want C.D == 5, got %d", C.D)
		}
	})

	t.Run("pointer to pointer to struct", func(t *testing.T) {
		s := &struct {
			A string `fig:",required"`
		}{}

		errs := make(fieldErrors)
		fig.validateCollection(reflect.ValueOf(&s), errs, "")

		if len(errs) == 0 {
			t.Fatalf("expected error")
		}

		if len(errs) > 1 {
			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 1, len(errs), errs)
		}

		if _, ok := errs["A"]; !ok {
			t.Fatalf("want A in errs, got %+v", errs)
		}
	})

	t.Run("slice of slices", func(t *testing.T) {
		type A struct {
			B string `fig:",required"`
		}

		s := make([][]A, 1)
		s[0] = make([]A, 1)

		errs := make(fieldErrors)
		fig.validateCollection(reflect.ValueOf(&s), errs, "")

		if len(errs) == 0 {
			t.Fatalf("expected error")
		}

		if len(errs) > 1 {
			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 1, len(errs), errs)
		}

		if _, ok := errs["[0][0].B"]; !ok {
			t.Fatalf("want [0][0].B in errs, got %+v", errs)
		}
	})

	t.Run("interface with underlying basic type is no-op", func(t *testing.T) {
		x := 5
		var iter interface{} = x

		errs := make(fieldErrors)
		fig.validateCollection(reflect.ValueOf(iter), errs, "")

		if len(errs) > 0 {
			t.Fatalf("unexpected err: %v", errs)
		}
	})
}

func Test_fig_validateFieldWithTag(t *testing.T) {
	f := newDefaultFig()

	t.Run("returns nil if tag does not contain validation keys", func(t *testing.T) {
		var s []string

		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), "s")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if len(s) != 0 {
			t.Fatalf("slice changed: %v", err)
		}
	})

	t.Run("returns error on too many tag keys", func(t *testing.T) {
		x := 0

		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",required,default=5")
		if err == nil {
			t.Errorf("expected err")
		}

		if x != 0 {
			t.Fatalf("x changed: %d", x)
		}
	})

	t.Run("returns error on unexpected tag key", func(t *testing.T) {
		d := 0.5

		err := f.validateFieldWithTag(reflect.ValueOf(&d).Elem(), ",whatami")
		if err == nil {
			t.Errorf("expected err")
		}

		if d != 0.5 {
			t.Fatalf("d changed: %f", d)
		}
	})
}

func Test_fig_validateFieldWithTag_required(t *testing.T) {
	f := newDefaultFig()

	t.Run("returns error on zero value", func(t *testing.T) {
		var s []string

		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), ",required")
		if err == nil {
			t.Errorf("expected err")
		}
	})

	t.Run("returns nil on non-zero value", func(t *testing.T) {
		s := []string{"foo"}

		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), ",required")
		if err != nil {
			t.Fatalf("unexpected err, %v", err)
		}
	})
}

func Test_fig_validateFieldWithTag_default(t *testing.T) {
	f := newDefaultFig()

	t.Run("sets default with leading field name", func(t *testing.T) {
		b := false

		err := f.validateFieldWithTag(reflect.ValueOf(&b).Elem(), "b,default=true")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if b != true {
			t.Fatalf("want %t, got %t", true, b)
		}
	})

	t.Run("sets default on zero value", func(t *testing.T) {
		x := 0

		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=5")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if x != 5 {
			t.Fatalf("want %d, got %d", 5, x)
		}
	})

	t.Run("does not set default on non-zero value", func(t *testing.T) {
		x := 1

		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=5")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if x != 1 {
			t.Fatalf("want %d, got %d", 1, x)
		}
	})

	t.Run("invalid default value returns error", func(t *testing.T) {
		x := 0

		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=notAnInt")
		if err == nil {
			t.Errorf("expected err")
		}

		if x != 0 {
			t.Fatalf("x changed: %v", x)
		}
	})

	t.Run("sets default time with custom layout", func(t *testing.T) {
		f := newDefaultFig()
		f.timeLayout = "01-2006"

		dt := time.Time{}

		err := f.validateFieldWithTag(reflect.ValueOf(&dt).Elem(), ",default=12-2019")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		want := time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC)
		if !want.Equal(dt) {
			t.Fatalf("want %v, got %v", want, dt)
		}
	})
}

func Test_fig_isZero(t *testing.T) {
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

func Test_fig_splitTagCommas(t *testing.T) {
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
			got := newDefaultFig().splitTagCommas(tc.S)

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

func Test_fig_setFieldValue(t *testing.T) {
	fig := newDefaultFig()

	t.Run("nil ptr", func(t *testing.T) {
		var s *string
		fv := reflect.ValueOf(&s)

		err := fig.setFieldValue(fv, "bat")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if *s != "bat" {
			t.Fatalf("want %s, got %s", "bat", *s)
		}
	})

	t.Run("slice", func(t *testing.T) {
		var slice []bool
		fv := reflect.ValueOf(&slice).Elem()

		err := fig.setFieldValue(fv, "true")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if !reflect.DeepEqual([]bool{true}, slice) {
			t.Fatalf("want %+v, got %+v", []bool{true}, slice)
		}
	})

	t.Run("bool", func(t *testing.T) {
		var b bool
		fv := reflect.ValueOf(&b).Elem()

		err := fig.setFieldValue(fv, "true")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if b == false {
			t.Fatalf("b != true")
		}
	})

	t.Run("int", func(t *testing.T) {
		var i int
		fv := reflect.ValueOf(&i).Elem()

		err := fig.setFieldValue(fv, "-8")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if i != -8 {
			t.Fatalf("want %d, got %d", -8, i)
		}
	})

	t.Run("duration", func(t *testing.T) {
		var d time.Duration
		fv := reflect.ValueOf(&d).Elem()

		err := fig.setFieldValue(fv, "5h")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if d.Hours() != 5 {
			t.Fatalf("want %v, got %v", 5*time.Hour, d)
		}
	})

	t.Run("uint", func(t *testing.T) {
		var i uint
		fv := reflect.ValueOf(&i).Elem()

		err := fig.setFieldValue(fv, "42")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if i != 42 {
			t.Fatalf("want %d, got %d", 42, i)
		}
	})

	t.Run("float", func(t *testing.T) {
		var f float32
		fv := reflect.ValueOf(&f).Elem()

		err := fig.setFieldValue(fv, "0.015625")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if f != 0.015625 {
			t.Fatalf("want %f, got %f", 0.015625, f)
		}
	})

	t.Run("string", func(t *testing.T) {
		var s string
		fv := reflect.ValueOf(&s).Elem()

		err := fig.setFieldValue(fv, "bat")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if s != "bat" {
			t.Fatalf("want %s, got %s", "bat", s)
		}
	})

	t.Run("time", func(t *testing.T) {
		var tme time.Time
		fv := reflect.ValueOf(&tme).Elem()

		err := fig.setFieldValue(fv, "2020-01-01T00:00:00Z")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		want, err := time.Parse(fig.timeLayout, "2020-01-01T00:00:00Z")
		if err != nil {
			t.Fatalf("error parsing time: %v", err)
		}

		if !tme.Equal(want) {
			t.Fatalf("want %v, got %v", want, tme)
		}
	})

	t.Run("interface returns error", func(t *testing.T) {
		var i interface{}
		fv := reflect.ValueOf(i)

		err := fig.setFieldValue(fv, "empty")
		if err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("struct returns error", func(t *testing.T) {
		s := struct{ Name string }{}
		fv := reflect.ValueOf(&s).Elem()

		err := fig.setFieldValue(fv, "foo")
		if err == nil {
			t.Fatalf("expected err")
		}
	})
}

func Test_fig_setSliceValue(t *testing.T) {
	f := newDefaultFig()

	for _, tc := range []struct {
		Name      string
		InSlice   interface{}
		WantSlice interface{}
		Val       string
	}{
		{
			Name:      "ints",
			InSlice:   &[]int{},
			WantSlice: &[]int{5, 10, 15},
			Val:       string(f.sliceStart) + "5,10,15" + string(f.sliceEnd),
		},
		{
			Name:      "uints",
			InSlice:   &[]uint{},
			WantSlice: &[]uint{5, 10, 15, 20, 25},
			Val:       string(f.sliceStart) + "5,10,15,20,25" + string(f.sliceEnd),
		},
		{
			Name:      "floats",
			InSlice:   &[]float32{},
			WantSlice: &[]float32{1.5, 1.125, -0.25},
			Val:       string(f.sliceStart) + "1.5,1.125,-0.25" + string(f.sliceEnd),
		},
		{
			Name:      "strings",
			InSlice:   &[]string{},
			WantSlice: &[]string{"a", "b", "c", "d"},
			Val:       string(f.sliceStart) + "a,b,c,d" + string(f.sliceEnd),
		},
		{
			Name:      "bools",
			InSlice:   &[]bool{},
			WantSlice: &[]bool{true, true, false, false, false, true},
			Val:       string(f.sliceStart) + "1,1,false,0,false,true" + string(f.sliceEnd),
		},
		{
			Name:      "durations",
			InSlice:   &[]time.Duration{},
			WantSlice: &[]time.Duration{30 * time.Minute, 2 * time.Hour},
			Val:       string(f.sliceStart) + "30m,2h" + string(f.sliceEnd),
		},
		{
			Name:    "times",
			InSlice: &[]time.Time{},
			WantSlice: &[]time.Time{
				time.Date(2019, 12, 25, 10, 30, 30, 0, time.UTC),
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Val: string(f.sliceStart) + "2019-12-25T10:30:30Z,2020-01-01T00:00:00Z" + string(f.sliceEnd),
		},
	} {
		t.Run(tc.Val, func(t *testing.T) {
			in := reflect.ValueOf(tc.InSlice).Elem()

			err := f.setSliceValue(in, tc.Val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			want := reflect.ValueOf(tc.WantSlice).Elem()

			if !reflect.DeepEqual(want.Interface(), in.Interface()) {
				t.Fatalf("want %+v, got %+v", want, in)
			}
		})
	}

	t.Run("negative int into uint returns error", func(t *testing.T) {
		in := &[]uint{}
		val := string(f.sliceStart) + "-5" + string(f.sliceEnd)

		err := f.setSliceValue(reflect.ValueOf(in).Elem(), val)
		if err == nil {
			t.Fatalf("expected err")
		}
	})
}

func Test_fig_stringSlice(t *testing.T) {
	f := newDefaultFig()

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
			In:   string(f.sliceStart) + "hello , world" + string(f.sliceEnd),
			Want: []string{"hello ", " world"},
		},
		{
			In:   string(f.sliceStart) + "foo" + string(f.sliceEnd),
			Want: []string{"foo"},
		},
	} {
		t.Run(tc.In, func(t *testing.T) {
			got := f.stringSlice(tc.In)
			if !reflect.DeepEqual(tc.Want, got) {
				t.Fatalf("want %+v, got %+v", tc.Want, got)
			}
		})
	}
}
