package fig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func Test_fig_replaceEnvironments(t *testing.T) {
	os.Setenv("FOO", "XXX")
	os.Setenv("BAR", "YYY")

	tests := []struct {
		name     string
		text     string
		want     string
		hasError bool
	}{
		{name: "environment with default value", text: "/x/y/${BAZ:a}", want: "/x/y/a"},
		{name: "no environment tag", text: "/x/y/z", want: "/x/y/z"},
		{name: "from environment", text: "/x/y/${FOO}", want: "/x/y/XXX"},
		{name: "environment when is not set", text: "/x/y/${BAZ}", want: "/x/y/"},
		{name: "environment when is not set and default value is missing", text: "/x/y/${BAZ:}", want: "/x/y/"},
		{name: "environment name is missing", text: "/x/y/${}", hasError: true},
		{name: "multiple environment names", text: "/x/y/${FOO}/z/${BAR}", want: "/x/y/XXX/z/YYY"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result, err := replaceEnvironments(test.text); err != nil {
				if test.hasError && err == nil {
					t.Error("not expected")
				}
			} else if test.want != result {
				t.Error("not expected")
			}
		})
	}
}

func Test_fig_Load_If_Env_Set_In_Conf_File(t *testing.T) {
	os.Setenv("POD_NAME", "ehcache")
	for _, f := range []string{"pod.yaml", "pod.json", "pod.toml"} {
		t.Run(f, func(t *testing.T) {
			var cfg Pod
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			want := validPodConfig()
			want.Metadata.Name = "ehcache"

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
						LogLevel   string `fig:"log_level" default:"info"`
						Production bool   `fig:"production"`
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

func Test_fig_Load_Server_If_Env_Set_In_Conf_File(t *testing.T) {
	os.Setenv("SERVICE_HOST", "192.168.0.128")
	for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
		t.Run(f, func(t *testing.T) {
			type Server struct {
				Host string `fig:"host"`
			}

			var cfg Server
			err := Load(&cfg, File(f), Dirs(filepath.Join("testdata", "valid")))
			if err != nil {
				t.Fatalf("expected err")
			}

			want := Server{Host: "192.168.0.128"}

			if !reflect.DeepEqual(want, cfg) {
				t.Errorf("\nwant %+v\ngot %+v", want, cfg)
			}
		})
	}
}

func Test_fig_Load_Server_With_Profile(t *testing.T) {
	for _, f := range []string{"server.yaml", "server.json", "server.toml"} {
		t.Run(f, func(t *testing.T) {
			fmt.Println(f)
			type Server struct {
				Host   string `fig:"host"`
				Logger struct {
					LogLevel string `fig:"log_level" default:"info"`
					Appender string `fig:"appender"`
				}
				Replicas []string
			}

			var cfg Server
			err := Load(&cfg,
				File(f),
				Dirs(filepath.Join("testdata", "valid")),
				Profiles("test"),
				ProfileLayout("config.test.yaml"),
			)
			if err != nil {
				t.Fatalf("expected err %v", err)
			}

			want := Server{Host: "192.168.0.256"}
			want.Logger.LogLevel = "error"
			want.Logger.Appender = "file"
			want.Replicas = []string{"xyz"}

			if !reflect.DeepEqual(want, cfg) {
				t.Errorf("\nwant %+v\ngot %+v", want, cfg)
			}
		})
	}
}

func Test_fig_Load_Server_With_Profile_When_Config_Is_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		profile string
	}{
		{name: "profile file is not found", file: "pod.yaml", profile: "test"},
		{name: "config file when bad format", file: "bad.yaml", profile: "test"},
		{name: "profile file when bad format", file: "pod.yaml", profile: "bad"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := struct{}{}
			err := Load(&cfg,
				File(test.file),
				Dirs(
					filepath.Join("testdata", "valid"),
					filepath.Join("testdata", "invalid"),
				),
				Profiles(test.profile),
				ProfileLayout("config-test.yaml"),
			)

			if err == nil {
				t.Fatalf("expected err %v", err)
			}
		})
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

func Test_fig_findProfileCfgFile(t *testing.T) {
	t.Run("finds existing file", func(t *testing.T) {
		fig := defaultFig()
		fig.filename = "server.yaml"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findProfileCfgFile("test")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		want := filepath.Join("testdata", "valid", "server.test.yaml")
		if want != file {
			t.Fatalf("want file %s, got %s", want, file)
		}
	})

	t.Run("non-existing file returns ErrFileNotFound", func(t *testing.T) {
		fig := defaultFig()
		fig.filename = "server.yaml"
		fig.dirs = []string{".", "testdata", filepath.Join("testdata", "valid")}

		file, err := fig.findProfileCfgFile("e2e")
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
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("os.Setenv() unexpected error: %v", err)
	}
}

// func Test_fig_validate(t *testing.T) {
// 	fig := defaultFig()
// 	fig.tag = "fig"
//
// 	t.Run("struct without tags does nothing", func(t *testing.T) {
// 		var cfg struct {
// 			A string
// 			B int
// 		}
//
// 		cfg.A = "foo"
// 		cfg.B = 9
//
// 		err := fig.validate(&cfg)
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		if cfg.A != "foo" {
// 			t.Fatalf("cfg.A: want %s, got %s", "foo", cfg.A)
// 		}
//
// 		if cfg.B != 9 {
// 			t.Fatalf("cfg.B: want %d, got %d", 9, cfg.B)
// 		}
// 	})
//
// 	t.Run("non pointer struct returns error", func(t *testing.T) {
// 		var cfg struct {
// 			A string `fig:"a,default=foo"`
// 		}
//
// 		err := fig.validate(cfg)
// 		if err == nil {
// 			t.Fatal("expected err")
// 		}
// 	})
//
// 	t.Run("returns field errors as error", func(t *testing.T) {
// 		var cfg struct {
// 			A string  `fig:"a,required"`
// 			D float32 `fig:"d,default=true"`
// 		}
//
// 		err := fig.validate(&cfg)
// 		if err == nil {
// 			t.Fatal("expected err")
// 		}
//
// 		fieldErrs, ok := err.(fieldErrors)
// 		if !ok {
// 			t.Fatalf("want err type %T, got %T; val %v", fieldErrors{}, err, err)
// 		}
//
// 		if len(fieldErrs) != 2 {
// 			t.Fatalf("want %d err, got %d; val %+v", 2, len(fieldErrs), fieldErrs)
// 		}
// 	})
// }
//
// func Test_fig_validateStruct(t *testing.T) {
// 	fig := defaultFig()
// 	fig.tag = "fig"
//
// 	t.Run("struct nil ptr ptr validated", func(t *testing.T) {
// 		type A struct {
// 			B string  `fig:",required"`
// 			X float32 `fig:",default=not-a-float"`
// 		}
//
// 		C := struct {
// 			A ***A `fig:",required"`
// 		}{}
//
// 		errs := make(fieldErrors)
// 		fig.validateStruct(reflect.ValueOf(&C).Elem(), errs, "")
//
// 		if len(errs) != 1 {
// 			t.Fatalf("want len(errs) == 1, got %d\nerrs == %+v", len(errs), errs)
// 		}
//
// 		if _, ok := errs["A"]; !ok {
// 			t.Fatalf("want A in errs, got %v", errs)
// 		}
// 	})
//
// 	t.Run("struct non-nil ptr inner fields validated", func(t *testing.T) {
// 		type A struct {
// 			B string  `fig:",required"`
// 			X float32 `fig:",default=not-a-float"`
// 		}
//
// 		C := struct {
// 			A **A
// 		}{}
//
// 		a := &A{}
// 		C.A = &a
//
// 		errs := make(fieldErrors)
// 		fig.validateStruct(reflect.ValueOf(&C).Elem(), errs, "")
//
// 		if len(errs) != 2 {
// 			t.Fatalf("want len(errs) == 2, got %d\nerrs == %+v", len(errs), errs)
// 		}
//
// 		if _, ok := errs["A.B"]; !ok {
// 			t.Fatalf("want A.B in errs, got %v", errs)
// 		}
//
// 		if _, ok := errs["A.X"]; !ok {
// 			t.Fatalf("want A.X in errs, got %v", errs)
// 		}
// 	})
//
// 	t.Run("nested structs validated", func(t *testing.T) {
// 		var test struct {
// 			A string `fig:",required"`
// 			B struct {
// 				C int `fig:",default=5"`
// 				D struct {
// 					E *float32 `fig:",default=0.125"`
// 				}
// 			}
// 		}
//
// 		test.A = "foo"
//
// 		var (
// 			fv   = reflect.ValueOf(&test).Elem()
// 			errs = make(fieldErrors)
// 			name = "test"
// 		)
//
// 		fig.validateStruct(fv, errs, name)
// 		if len(errs) > 0 {
// 			t.Fatalf("unexpected err: %v", errs)
// 		}
//
// 		if test.A != "foo" {
// 			t.Fatalf("test.A: want %s, got %s", "foo", test.A)
// 		}
//
// 		if test.B.C != 5 {
// 			t.Fatalf("test.B.C: want %d, got %d", 5, test.B.C)
// 		}
//
// 		if *test.B.D.E != 0.125 {
// 			t.Fatalf("test.B.D.E: want %fig, got %fig", 0.125, *test.B.D.E)
// 		}
// 	})
//
// 	t.Run("slice field names set", func(t *testing.T) {
// 		type A struct {
// 			B string `fig:",required"`
// 		}
//
// 		type C struct {
// 			As []A `fig:"required"`
// 		}
//
// 		type I struct {
// 			X int `fig:",required"`
// 		}
//
// 		D := struct {
// 			*I
// 			Cs []C `fig:"required"`
// 		}{}
//
// 		D.I = &I{}
// 		D.Cs = []C{
// 			{
// 				As: []A{{}, {}},
// 			},
// 		}
//
// 		errs := make(fieldErrors)
// 		fig.validateStruct(reflect.ValueOf(D), errs, "")
//
// 		if len(errs) == 0 {
// 			t.Fatalf("expected err")
// 		}
//
// 		if len(errs) != 3 {
// 			t.Fatalf("expected len(errs) == 3, got %d\nerrs = %+v", len(errs), errs)
// 		}
//
// 		wants := []string{"I.X", "Cs[0].As[0].B", "Cs[0].As[1].B"}
// 		for _, want := range wants {
// 			if _, ok := errs[want]; !ok {
// 				t.Errorf("want %s in errs, got %+v", want, errs)
// 			}
// 		}
// 	})
//
// 	t.Run("returns all field errors", func(t *testing.T) {
// 		var test struct {
// 			A string `fig:",required"`
// 			B struct {
// 				C int `fig:",badkey"`
// 				D *struct {
// 					E *float32 `fig:",required"`
// 				}
// 				I interface{} `fig:",default=5"`
// 				S string      `fig:",required"`
// 			}
// 		}
//
// 		test.B.S = "ok"
//
// 		var (
// 			fv   = reflect.ValueOf(&test).Elem()
// 			errs = make(fieldErrors)
// 			name = "test"
// 		)
//
// 		fig.validateStruct(fv, errs, name)
// 		if len(errs) == 0 {
// 			t.Fatal("expected err")
// 		}
//
// 		// test.B.D.E not reported as an error as *D is nil
// 		wantErrs := []string{"test.A", "test.B.C", "test.B.I"}
// 		if len(wantErrs) != len(errs) {
// 			t.Fatalf("want %d errs, got %d", len(wantErrs), len(errs))
// 		}
//
// 		for _, want := range wantErrs {
// 			if _, ok := errs[want]; !ok {
// 				t.Fatalf("want %s in errs, instead contains %+v", want, errs)
// 			}
// 		}
// 	})
// }
//
// func Test_fig_validateField(t *testing.T) {
// 	fig := defaultFig()
// 	fig.tag = "fig"
//
// 	t.Run("nil struct does not validate inner fields", func(t *testing.T) {
// 		A := struct {
// 			B *struct {
// 				C string `fig:"C,required"`
// 				D bool   `fig:"D,required"`
// 			}
// 		}{}
//
// 		fv := reflect.ValueOf(A).Field(0)
// 		fd := reflect.ValueOf(A).Type().Field(0)
//
// 		errs := make(fieldErrors)
// 		fig.validateFieldTag(fv, fd, errs, "")
//
// 		if len(errs) > 0 {
// 			t.Fatalf("unexpected err: %v", errs)
// 		}
// 	})
//
// 	t.Run("struct wrapped in interface validated", func(t *testing.T) {
// 		A := struct {
// 			I interface{}
// 		}{}
//
// 		C := struct {
// 			D string `fig:",required"`
// 			E int    `fig:",default=5"`
// 		}{}
//
// 		A.I = &C
//
// 		fv := reflect.ValueOf(A).Field(0)
// 		fd := reflect.ValueOf(A).Type().Field(0)
//
// 		errs := make(fieldErrors)
// 		fig.validateFieldTag(fv, fd, errs, "")
//
// 		if len(errs) != 1 {
// 			t.Fatalf("want len(errs) == 1, got %d\nerrs = %+v", len(errs), errs)
// 		}
//
// 		if _, ok := errs["I.D"]; !ok {
// 			t.Fatalf("want I.D in errs, got %+v\n", errs)
// 		}
//
// 		if C.E != 5 {
// 			t.Fatalf("want C.E == 5, got %d", C.E)
// 		}
// 	})
// }
//
// func Test_fig_validateCollection(t *testing.T) {
// 	fig := defaultFig()
// 	fig.tag = "fig"
//
// 	t.Run("slice of struct ptrs", func(t *testing.T) {
// 		type A struct {
// 			B string `fig:",required"`
// 			S []int  `fig:",required"`
// 		}
//
// 		C := struct {
// 			As []*A
// 		}{
// 			As: []*A{
// 				{},
// 			},
// 		}
//
// 		errs := make(fieldErrors)
// 		fig.validateCollection(reflect.ValueOf(&C), errs, "")
//
// 		if len(errs) == 0 {
// 			t.Fatalf("expected error")
// 		}
//
// 		if len(errs) != 2 {
// 			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 2, len(errs), errs)
// 		}
//
// 		for _, want := range []string{"As[0].B", "As[0].S"} {
// 			if _, ok := errs[want]; !ok {
// 				t.Fatalf("want %s in errs, got %+v", want, errs)
// 			}
// 		}
// 	})
//
// 	t.Run("anonymous struct", func(t *testing.T) {
// 		type A struct {
// 			B string `fig:",required"`
// 			D int    `fig:",default=5"`
// 		}
//
// 		C := struct {
// 			A
// 		}{}
//
// 		errs := make(fieldErrors)
// 		fig.validateCollection(reflect.ValueOf(&C).Elem(), errs, "")
//
// 		if len(errs) != 1 {
// 			t.Fatalf("want len(errs) == 1, got %d\nerrs = %+v", len(errs), errs)
// 		}
//
// 		if _, ok := errs["A.B"]; !ok {
// 			t.Errorf("want A.B in errs, got %+v", errs)
// 		}
//
// 		if C.D != 5 {
// 			t.Errorf("want C.D == 5, got %d", C.D)
// 		}
// 	})
//
// 	t.Run("pointer to pointer to struct", func(t *testing.T) {
// 		s := &struct {
// 			A string `fig:",required"`
// 		}{}
//
// 		errs := make(fieldErrors)
// 		fig.validateCollection(reflect.ValueOf(&s), errs, "")
//
// 		if len(errs) == 0 {
// 			t.Fatalf("expected error")
// 		}
//
// 		if len(errs) > 1 {
// 			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 1, len(errs), errs)
// 		}
//
// 		if _, ok := errs["A"]; !ok {
// 			t.Fatalf("want A in errs, got %+v", errs)
// 		}
// 	})
//
// 	t.Run("slice of slices", func(t *testing.T) {
// 		type A struct {
// 			B string `fig:",required"`
// 		}
//
// 		s := make([][]A, 1)
// 		s[0] = make([]A, 1)
//
// 		errs := make(fieldErrors)
// 		fig.validateCollection(reflect.ValueOf(&s), errs, "")
//
// 		if len(errs) == 0 {
// 			t.Fatalf("expected error")
// 		}
//
// 		if len(errs) > 1 {
// 			t.Fatalf("want len(errs) == %d, got %d\nerrs = %+v", 1, len(errs), errs)
// 		}
//
// 		if _, ok := errs["[0][0].B"]; !ok {
// 			t.Fatalf("want [0][0].B in errs, got %+v", errs)
// 		}
// 	})
//
// 	t.Run("interface with underlying basic type is no-op", func(t *testing.T) {
// 		x := 5
// 		var iter interface{} = x
//
// 		errs := make(fieldErrors)
// 		fig.validateCollection(reflect.ValueOf(iter), errs, "")
//
// 		if len(errs) > 0 {
// 			t.Fatalf("unexpected err: %v", errs)
// 		}
// 	})
// }

// func Test_fig_validateFieldWithTag(t *testing.T) {
// 	f := defaultFig()
//
// 	t.Run("returns nil if tag does not contain validation keys", func(t *testing.T) {
// 		var s []string
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), "s")
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		if len(s) != 0 {
// 			t.Fatalf("slice changed: %v", err)
// 		}
// 	})
//
// 	t.Run("returns error on too many tag keys", func(t *testing.T) {
// 		x := 0
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",required,default=5")
// 		if err == nil {
// 			t.Errorf("expected err")
// 		}
//
// 		if x != 0 {
// 			t.Fatalf("x changed: %d", x)
// 		}
// 	})
//
// 	t.Run("returns error on unexpected tag key", func(t *testing.T) {
// 		d := 0.5
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&d).Elem(), ",whatami")
// 		if err == nil {
// 			t.Errorf("expected err")
// 		}
//
// 		if d != 0.5 {
// 			t.Fatalf("d changed: %f", d)
// 		}
// 	})
// }
//
// func Test_fig_validateFieldWithTag_required(t *testing.T) {
// 	f := defaultFig()
//
// 	t.Run("returns error on zero value", func(t *testing.T) {
// 		var s []string
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), ",required")
// 		if err == nil {
// 			t.Errorf("expected err")
// 		}
// 	})
//
// 	t.Run("returns nil on non-zero value", func(t *testing.T) {
// 		s := []string{"foo"}
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), ",required")
// 		if err != nil {
// 			t.Fatalf("unexpected err, %v", err)
// 		}
// 	})
// }
//
// func Test_fig_validateFieldWithTag_default(t *testing.T) {
// 	f := defaultFig()
//
// 	t.Run("sets default with leading field name", func(t *testing.T) {
// 		var s string
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&s).Elem(), "b,default=hey")
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		if s != "hey" {
// 			t.Fatalf("want %s, got %s", "hey", s)
// 		}
// 	})
//
// 	t.Run("sets default on zero value", func(t *testing.T) {
// 		x := 0
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=5")
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		if x != 5 {
// 			t.Fatalf("want %d, got %d", 5, x)
// 		}
// 	})
//
// 	t.Run("does not set default on non-zero value", func(t *testing.T) {
// 		x := 1
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=5")
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		if x != 1 {
// 			t.Fatalf("want %d, got %d", 1, x)
// 		}
// 	})
//
// 	t.Run("invalid default value returns error", func(t *testing.T) {
// 		x := 0
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&x).Elem(), ",default=notAnInt")
// 		if err == nil {
// 			t.Errorf("expected err")
// 		}
//
// 		if x != 0 {
// 			t.Fatalf("x changed: %v", x)
// 		}
// 	})
//
// 	t.Run("sets default time with custom layout", func(t *testing.T) {
// 		f := defaultFig()
// 		f.timeLayout = "01-2006"
//
// 		dt := time.Time{}
//
// 		err := f.validateFieldWithTag(reflect.ValueOf(&dt).Elem(), ",default=12-2019")
// 		if err != nil {
// 			t.Fatalf("unexpected err: %v", err)
// 		}
//
// 		want := time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC)
// 		if !want.Equal(dt) {
// 			t.Fatalf("want %v, got %v", want, dt)
// 		}
// 	})
// }
