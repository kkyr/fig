package fig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type Pod struct {
	APIVersion string `fig:"apiVersion" default:"v1"`
	Kind       string `fig:"kind" validate:"required"`
	Metadata   struct {
		Name           string        `fig:"name"`
		Environments   []string      `fig:"environments" default:"[dev,staging,prod]"`
		Master         bool          `fig:"master" validate:"required"`
		MaxPercentUtil *float64      `fig:"maxPercentUtil" default:"0.5"`
		Retry          time.Duration `fig:"retry" default:"10s"`
	} `fig:"metadata"`
	Spec Spec `fig:"spec"`
}

type Spec struct {
	Containers []Container `fig:"containers"`
	Volumes    []*Volume   `fig:"volumes"`
}

type Container struct {
	Name      string   `fig:"name" validate:"required"`
	Image     string   `fig:"image" validate:"required"`
	Command   []string `fig:"command"`
	Env       []Env    `fig:"env"`
	Ports     []Port   `fig:"ports"`
	Resources struct {
		Limits struct {
			CPU string `fig:"cpu"`
		} `fig:"limits"`
		Requests *struct {
			Memory string  `fig:"memory" default:"64Mi"`
			CPU    *string `fig:"cpu" default:"250m"`
		}
	} `fig:"resources"`
	VolumeMounts []VolumeMount `fig:"volumeMounts"`
}

type Env struct {
	Name  string `fig:"name"`
	Value string `fig:"value"`
}

type Port struct {
	ContainerPort int `fig:"containerPort" validate:"required"`
}

type VolumeMount struct {
	MountPath string `fig:"mountPath" validate:"required"`
	Name      string `fig:"name" validate:"required"`
}

type Volume struct {
	Name      string     `fig:"name" validate:"required"`
	ConfigMap *ConfigMap `fig:"configMap"`
}

type ConfigMap struct {
	Name  string `fig:"name" validate:"required"`
	Items []Item `fig:"items" validate:"required"`
}

type Item struct {
	Key  string `fig:"key" validate:"required"`
	Path string `fig:"path" validate:"required"`
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

func Test_fig_Load_FileNotFound(t *testing.T) {
	fig := defaultFig()
	fig.filename = "abrakadabra"
	var cfg Pod
	err := fig.Load(&cfg)
	if err == nil {
		t.Fatalf("expected err")
	}
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected err %v, got %v", ErrFileNotFound, err)
	}
}

func Test_fig_Load_NonStructPtr(t *testing.T) {
	cfg := struct {
		X int
	}{}
	fig := defaultFig()
	err := fig.Load(cfg)
	if err == nil {
		t.Fatalf("fig.Load() returned nil error")
	}
	if !strings.Contains(err.Error(), "pointer") {
		t.Errorf("expected struct pointer err, got %v", err)
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
				"kind",
				"metadata.master",
				"spec.containers[0].image",
				"spec.volumes[0].configMap.items",
				"spec.volumes[1].name",
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
					Host   string `fig:"host" default:"127.0.0.1"`
					Ports  []int  `fig:"ports" default:"[80,443]"`
					Logger struct {
						LogLevel   string         `fig:"log_level" default:"info"`
						Pattern    *regexp.Regexp `fig:"pattern" default:".*"`
						Production bool           `fig:"production"`
						Metadata   struct {
							Keys []string `fig:"keys" default:"[ts]"`
						}
					}
					Application struct {
						BuildDate time.Time `fig:"build_date" default:"2020-01-01T12:00:00Z"`
					}
				}

				var want Server
				want.Host = "0.0.0.0"
				want.Ports = []int{80, 443}
				want.Logger.LogLevel = "debug"
				want.Logger.Pattern = regexp.MustCompile(".*")
				want.Logger.Production = false
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
					Host   string `fig:"host" default:"127.0.0.1"`
					Ports  []int  `fig:"ports" default:"[80,not-a-port]"`
					Logger struct {
						LogLevel string `fig:"log_level" default:"info"`
						Metadata struct {
							Keys []string `fig:"keys" validate:"required"`
						}
					}
					Application struct {
						BuildDate time.Time `fig:"build_date" default:"not-a-time"`
					}
				}

				var cfg Server
				err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
				if err == nil {
					t.Fatalf("expected err")
				}

				want := []string{
					"ports",
					"Logger.Metadata.keys",
					"Application.build_date",
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
				Host   string `fig:"host" default:"127.0.0.1"`
				Ports  []int  `fig:"ports" validate:"required"`
				Logger struct {
					LogLevel string `fig:"log_level" validate:"required"`
					Metadata struct {
						Keys []string `fig:"keys" validate:"required"`
					}
				}
				Application struct {
					BuildDate time.Time `fig:"build_date" default:"2020-01-01T12:00:00Z"`
				}
			}

			var cfg Server
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
			if err == nil {
				t.Fatalf("expected err")
			}

			want := []string{
				"ports",
				"Logger.Metadata.keys",
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

func Test_fig_Load_WithOptions(t *testing.T) {
	for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
		t.Run(f, func(t *testing.T) {
			type Server struct {
				Host   string `custom:"host" default:"127.0.0.1"`
				Ports  []int  `custom:"ports" default:"[80,443]"`
				Logger struct {
					LogLevel string `custom:"log_level"`
					Metadata struct {
						Keys []string `custom:"keys" default:"ts"`
						Tag  string   `custom:"tag" validate:"required"`
					}
				}
				Cache struct {
					CleanupInterval time.Duration `custom:"cleanup_interval" validate:"required"`
					FillThreshold   float32       `custom:"threshold" default:"0.9"`
				}
				Application struct {
					BuildDate time.Time `custom:"build_date" default:"12-25-2012"`
					Version   int
				}
			}

			os.Clearenv()
			setenv(t, "MYAPP_LOGGER_METADATA_TAG", "errorLogger")
			setenv(t, "MYAPP_LOGGER_LOG_LEVEL", "error")
			setenv(t, "MYAPP_APPLICATION_VERSION", "1")
			setenv(t, "MYAPP_CACHE_CLEANUP_INTERVAL", "5m")
			setenv(t, "MYAPP_CACHE_THRESHOLD", "0.85")

			var want Server
			want.Host = "0.0.0.0"
			want.Ports = []int{80, 443}
			want.Logger.LogLevel = "error"
			want.Logger.Metadata.Keys = []string{"ts"}
			want.Application.BuildDate = time.Date(2012, 12, 25, 0, 0, 0, 0, time.UTC)
			want.Logger.Metadata.Tag = "errorLogger"
			want.Application.Version = 1
			want.Cache.CleanupInterval = 5 * time.Minute
			want.Cache.FillThreshold = 0.85

			var cfg Server

			err := Load(&cfg,
				File(f),
				Dirs(filepath.Join("testdata", "valid")),
				Tag("custom"),
				TimeLayout("01-02-2006"),
				UseEnv("myapp"),
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

func Test_fig_Load_IgnoreFile(t *testing.T) {
	type Server struct {
		Host   string `custom:"host" default:"127.0.0.1"`
		Ports  []int  `custom:"ports" default:"[80,443]"`
		Logger struct {
			LogLevel string `custom:"log_level"`
			Metadata struct {
				Keys []string `custom:"keys" default:"ts"`
				Tag  string   `custom:"tag" validate:"required"`
			}
		}
		Cache struct {
			CleanupInterval time.Duration `custom:"cleanup_interval" validate:"required"`
			FillThreshold   float32       `custom:"threshold" default:"0.9"`
		}
		Application struct {
			BuildDate time.Time `custom:"build_date" default:"12-25-2012"`
			Version   int
		}
	}

	os.Clearenv()
	setenv(t, "MYAPP_HOST", "0.0.0.0")
	setenv(t, "MYAPP_PORTS", "[8888,443]")
	setenv(t, "MYAPP_LOGGER_METADATA_TAG", "errorLogger")
	setenv(t, "MYAPP_LOGGER_LOG_LEVEL", "error")
	setenv(t, "MYAPP_APPLICATION_VERSION", "1")
	setenv(t, "MYAPP_CACHE_CLEANUP_INTERVAL", "5m")
	setenv(t, "MYAPP_CACHE_THRESHOLD", "0.85")

	var want Server
	want.Host = "0.0.0.0"
	want.Ports = []int{8888, 443}
	want.Logger.LogLevel = "error"
	want.Logger.Metadata.Keys = []string{"ts"}
	want.Application.BuildDate = time.Date(2012, 12, 25, 0, 0, 0, 0, time.UTC)
	want.Logger.Metadata.Tag = "errorLogger"
	want.Application.Version = 1
	want.Cache.CleanupInterval = 5 * time.Minute
	want.Cache.FillThreshold = 0.85

	var cfg Server

	err := Load(&cfg,
		IgnoreFile(),
		Tag("custom"),
		TimeLayout("01-02-2006"),
		UseEnv("myapp"),
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if !reflect.DeepEqual(want, cfg) {
		t.Errorf("\nwant %+v\ngot %+v", want, cfg)
	}
}

func Test_fig_findCfgFile(t *testing.T) {
	t.Run("finds existing file", func(t *testing.T) {
		fig := defaultFig()
		fig.filename = "pod.yaml"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findCfgFile()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		want := filepath.Join("testdata", "valid", "pod.yaml")
		if want != file {
			t.Fatalf("want file %s, got %s", want, file)
		}
	})

	t.Run("non-existing file returns ErrFileNotFound", func(t *testing.T) {
		fig := defaultFig()
		fig.filename = "nope.nope"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findCfgFile()
		if err == nil {
			t.Fatalf("expected err, got file %s", file)
		}
		if !errors.Is(err, ErrFileNotFound) {
			t.Errorf("expected err %v, got %v", ErrFileNotFound, err)
		}
	})
}

func Test_fig_decodeFile(t *testing.T) {
	fig := defaultFig()

	for _, f := range []string{"bad.yaml", "bad.json", "bad.toml"} {
		t.Run(f, func(t *testing.T) {
			file := filepath.Join("testdata", "invalid", f)
			if !fileExists(file) {
				t.Fatalf("test file %s does not exist", file)
			}
			_, err := fig.decodeFile(file)
			if err == nil {
				t.Errorf("received nil error")
			}
		})
	}

	t.Run("unsupported file extension", func(t *testing.T) {
		file := filepath.Join("testdata", "invalid", "list.hcl")
		if !fileExists(file) {
			t.Fatalf("test file %s does not exist", file)
		}
		_, err := fig.decodeFile(file)
		if err == nil {
			t.Fatal("received nil error")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("err == %v, expected unsupported file extension", err)
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		_, err := fig.decodeFile("casperthefriendlygho.st")
		if err == nil {
			t.Fatal("received nil error")
		}
	})
}

func Test_fig_decodeMap(t *testing.T) {
	fig := defaultFig()
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
		Severity int    `fig:"severity" validate:"required"`
		Server   struct {
			Ports  []string `fig:"ports" default:"[443]"`
			Secure bool
		} `fig:"server"`
	}

	if err := fig.decodeMap(m, &cfg); err != nil {
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

func Test_fig_processCfg(t *testing.T) {
	t.Run("slice elements set by env", func(t *testing.T) {
		fig := defaultFig()
		fig.tag = "fig"
		fig.useEnv = true

		os.Clearenv()
		setenv(t, "A_0_B", "b0")
		setenv(t, "A_1_B", "b1")
		setenv(t, "A_0_C", "9000")

		cfg := struct {
			A []struct {
				B string `validate:"required"`
				C int    `default:"5"`
			}
		}{}
		cfg.A = []struct {
			B string `validate:"required"`
			C int    `default:"5"`
		}{{B: "boo"}, {B: "boo"}}

		err := fig.processCfg(&cfg)
		if err != nil {
			t.Fatalf("processCfg() returned unexpected error: %v", err)
		}
		if cfg.A[0].B != "b0" {
			t.Errorf("cfg.A[0].B == %s, expected %s", cfg.A[0].B, "b0")
		}
		if cfg.A[1].B != "b1" {
			t.Errorf("cfg.A[1].B == %s, expected %s", cfg.A[1].B, "b1")
		}
		if cfg.A[0].C != 9000 {
			t.Errorf("cfg.A[0].C == %d, expected %d", cfg.A[0].C, 9000)
		}
		if cfg.A[1].C != 5 {
			t.Errorf("cfg.A[1].C == %d, expected %d", cfg.A[1].C, 5)
		}
	})

	t.Run("embedded struct set by env", func(t *testing.T) {
		fig := defaultFig()
		fig.useEnv = true
		fig.tag = "fig"

		type A struct {
			B string
		}
		type C struct {
			D *int
		}
		type F struct {
			A
			C `fig:"cc"`
		}
		cfg := F{}

		os.Clearenv()
		setenv(t, "A_B", "embedded")
		setenv(t, "CC_D", "7")

		err := fig.processCfg(&cfg)
		if err != nil {
			t.Fatalf("processCfg() returned unexpected error: %v", err)
		}
		if cfg.A.B != "embedded" {
			t.Errorf("cfg.A.B == %s, expected %s", cfg.A.B, "embedded")
		}
		if *cfg.C.D != 7 {
			t.Errorf("cfg.C.D == %d, expected %d", *cfg.C.D, 7)
		}
	})
}

func Test_fig_processField(t *testing.T) {
	fig := defaultFig()
	fig.tag = "fig"

	t.Run("field with default", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" default:"10"`
		}{}
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err != nil {
			t.Fatalf("processField() returned unexpected error: %v", err)
		}
		if cfg.X != 10 {
			t.Errorf("cfg.X == %d, expected %d", cfg.X, 10)
		}
	})

	t.Run("field with default does not overwrite", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" default:"10"`
		}{}
		cfg.X = 5
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err != nil {
			t.Fatalf("processField() returned unexpected error: %v", err)
		}
		if cfg.X != 5 {
			t.Errorf("cfg.X == %d, expected %d", cfg.X, 5)
		}
	})

	t.Run("field with bad default", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" default:"not-an-int"`
		}{}
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err == nil {
			t.Fatalf("processField() returned nil error")
		}
	})

	t.Run("field with required", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" validate:"required"`
		}{}
		cfg.X = 10
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err != nil {
			t.Fatalf("processField() returned unexpected error: %v", err)
		}
		if cfg.X != 10 {
			t.Errorf("cfg.X == %d, expected %d", cfg.X, 10)
		}
	})

	t.Run("field with required error", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" validate:"required"`
		}{}
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err == nil {
			t.Fatalf("processField() returned nil error")
		}
	})

	t.Run("field with default and required", func(t *testing.T) {
		cfg := struct {
			X int `fig:"y" default:"10" validate:"required"`
		}{}
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err == nil {
			t.Fatalf("processField() expected error")
		}
	})

	t.Run("field overwritten by env", func(t *testing.T) {
		fig := defaultFig()
		fig.tag = "fig"
		fig.useEnv = true
		fig.envPrefix = "fig"

		os.Clearenv()
		setenv(t, "FIG_X", "MEN")

		cfg := struct {
			X string `fig:"x"`
		}{}
		cfg.X = "BOYS"
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err != nil {
			t.Fatalf("processField() returned unexpected error: %v", err)
		}
		if cfg.X != "MEN" {
			t.Errorf("cfg.X == %s, expected %s", cfg.X, "MEN")
		}
	})

	t.Run("field with bad env", func(t *testing.T) {
		fig := defaultFig()
		fig.tag = "fig"
		fig.useEnv = true
		fig.envPrefix = "fig"

		os.Clearenv()
		setenv(t, "FIG_I", "FIFTY")

		cfg := struct {
			I int
		}{}
		parent := &field{
			v:        reflect.ValueOf(&cfg).Elem(),
			t:        reflect.ValueOf(&cfg).Elem().Type(),
			sliceIdx: -1,
		}

		f := newStructField(parent, 0, fig.tag)
		err := fig.processField(f)
		if err == nil {
			t.Fatalf("processField() returned nil error")
		}
	})
}

func Test_fig_setFromEnv(t *testing.T) {
	fig := defaultFig()
	fig.envPrefix = "fig"

	var s string
	fv := reflect.ValueOf(&s)

	os.Clearenv()
	err := fig.setFromEnv(fv, "config.string")
	if err != nil {
		t.Fatalf("setFromEnv() unexpected error: %v", err)
	}
	if s != "" {
		t.Fatalf("s modified to %s", s)
	}

	setenv(t, "FIG_CONFIG_STRING", "goroutine")
	err = fig.setFromEnv(fv, "config.string")
	if err != nil {
		t.Fatalf("setFromEnv() unexpected error: %v", err)
	}
	if s != "goroutine" {
		t.Fatalf("s == %s, expected %s", s, "goroutine")
	}
}

func Test_fig_formatEnvKey(t *testing.T) {
	fig := defaultFig()

	for _, tc := range []struct {
		key    string
		prefix string
		want   string
	}{
		{
			key:  "port",
			want: "PORT",
		},
		{
			key:    "server.host",
			prefix: "myapp",
			want:   "MYAPP_SERVER_HOST",
		},
		{
			key:  "loggers[0].log_level",
			want: "LOGGERS_0_LOG_LEVEL",
		},
		{
			key:  "nested[1].slice[2].twice",
			want: "NESTED_1_SLICE_2_TWICE",
		},
		{
			key:    "client.http.timeout",
			prefix: "auth_s",
			want:   "AUTH_S_CLIENT_HTTP_TIMEOUT",
		},
	} {
		t.Run(fmt.Sprintf("%s/%s", tc.prefix, tc.key), func(t *testing.T) {
			fig.envPrefix = tc.prefix
			got := fig.formatEnvKey(tc.key)
			if got != tc.want {
				t.Errorf("formatEnvKey() == %s, expected %s", got, tc.want)
			}
		})
	}
}

func Test_fig_setDefaultValue(t *testing.T) {
	fig := defaultFig()
	var b bool
	fv := reflect.ValueOf(&b).Elem()

	err := fig.setDefaultValue(fv, "true")
	if err == nil {
		t.Fatalf("expected err")
	}
}

func Test_fig_setValue(t *testing.T) {
	fig := defaultFig()

	t.Run("nil ptr", func(t *testing.T) {
		var s *string
		fv := reflect.ValueOf(&s)

		err := fig.setValue(fv, "bat")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if *s != "bat" {
			t.Fatalf("want %s, got %s", "bat", *s)
		}
	})

	t.Run("slice", func(t *testing.T) {
		var slice []int
		fv := reflect.ValueOf(&slice).Elem()

		err := fig.setValue(fv, "5")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if !reflect.DeepEqual([]int{5}, slice) {
			t.Fatalf("want %+v, got %+v", []int{5}, slice)
		}
	})

	t.Run("int", func(t *testing.T) {
		var i int
		fv := reflect.ValueOf(&i).Elem()

		err := fig.setValue(fv, "-8")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if i != -8 {
			t.Fatalf("want %d, got %d", -8, i)
		}
	})

	t.Run("bool", func(t *testing.T) {
		var b bool
		fv := reflect.ValueOf(&b).Elem()

		err := fig.setValue(fv, "true")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if !b {
			t.Fatalf("want true")
		}
	})

	t.Run("bad bool", func(t *testing.T) {
		var b bool
		fv := reflect.ValueOf(&b).Elem()

		err := fig.setValue(fv, "αλήθεια")
		if err == nil {
			t.Fatalf("returned nil err")
		}
	})

	t.Run("duration", func(t *testing.T) {
		var d time.Duration
		fv := reflect.ValueOf(&d).Elem()

		err := fig.setValue(fv, "5h")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if d.Hours() != 5 {
			t.Fatalf("want %v, got %v", 5*time.Hour, d)
		}
	})

	t.Run("bad duration", func(t *testing.T) {
		var d time.Duration
		fv := reflect.ValueOf(&d).Elem()

		err := fig.setValue(fv, "5decades")
		if err == nil {
			t.Fatalf("expexted err")
		}
	})

	t.Run("uint", func(t *testing.T) {
		var i uint
		fv := reflect.ValueOf(&i).Elem()

		err := fig.setValue(fv, "42")
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

		err := fig.setValue(fv, "0.015625")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if f != 0.015625 {
			t.Fatalf("want %f, got %f", 0.015625, f)
		}
	})

	t.Run("bad float", func(t *testing.T) {
		var f float32
		fv := reflect.ValueOf(&f).Elem()

		err := fig.setValue(fv, "-i")
		if err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("string", func(t *testing.T) {
		var s string
		fv := reflect.ValueOf(&s).Elem()

		err := fig.setValue(fv, "bat")
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

		err := fig.setValue(fv, "2020-01-01T00:00:00Z")
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

	t.Run("bad time", func(t *testing.T) {
		var tme time.Time
		fv := reflect.ValueOf(&tme).Elem()

		err := fig.setValue(fv, "2020-Feb-01T00:00:00Z")
		if err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("regexp", func(t *testing.T) {
		var re regexp.Regexp
		fv := reflect.ValueOf(&re).Elem()

		err := fig.setValue(fv, "[a-z]+")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if want := regexp.MustCompile("[a-z]+"); re.String() != want.String() {
			t.Fatalf("want %v, got %v", want, re)
		}
	})

	t.Run("bad regexp", func(t *testing.T) {
		var re regexp.Regexp
		fv := reflect.ValueOf(&re).Elem()

		err := fig.setValue(fv, "[a-")
		if err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("interface returns error", func(t *testing.T) {
		var i interface{}
		fv := reflect.ValueOf(i)

		err := fig.setValue(fv, "empty")
		if err == nil {
			t.Fatalf("expected err")
		}
	})

	t.Run("struct returns error", func(t *testing.T) {
		s := struct{ Name string }{}
		fv := reflect.ValueOf(&s).Elem()

		err := fig.setValue(fv, "foo")
		if err == nil {
			t.Fatalf("expected err")
		}
	})
}

func Test_fig_setSlice(t *testing.T) {
	f := defaultFig()

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
			Val:       "[5,10,15]",
		},
		{
			Name:      "ints-no-square-braces",
			InSlice:   &[]int{},
			WantSlice: &[]int{5, 10, 15},
			Val:       "5,10,15",
		},
		{
			Name:      "uints",
			InSlice:   &[]uint{},
			WantSlice: &[]uint{5, 10, 15, 20, 25},
			Val:       "[5,10,15,20,25]",
		},
		{
			Name:      "floats",
			InSlice:   &[]float32{},
			WantSlice: &[]float32{1.5, 1.125, -0.25},
			Val:       "[1.5,1.125,-0.25]",
		},
		{
			Name:      "strings",
			InSlice:   &[]string{},
			WantSlice: &[]string{"a", "b", "c", "d"},
			Val:       "[a,b,c,d]",
		},
		{
			Name:      "durations",
			InSlice:   &[]time.Duration{},
			WantSlice: &[]time.Duration{30 * time.Minute, 2 * time.Hour},
			Val:       "[30m,2h]",
		},
		{
			Name:    "times",
			InSlice: &[]time.Time{},
			WantSlice: &[]time.Time{
				time.Date(2019, 12, 25, 10, 30, 30, 0, time.UTC),
				time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Val: "[2019-12-25T10:30:30Z,2020-01-01T00:00:00Z]",
		},
		{
			Name:    "regexps",
			InSlice: &[]*regexp.Regexp{},
			WantSlice: &[]*regexp.Regexp{
				regexp.MustCompile("[a-z]+"),
				regexp.MustCompile(".*"),
			},
			Val: "[[a-z]+,.*]",
		},
	} {
		t.Run(tc.Val, func(t *testing.T) {
			in := reflect.ValueOf(tc.InSlice).Elem()

			err := f.setSlice(in, tc.Val)
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
		val := "[-5]"

		err := f.setSlice(reflect.ValueOf(in).Elem(), val)
		if err == nil {
			t.Fatalf("expected err")
		}
	})
}

func setenv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}
