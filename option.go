package fig

// Option configures how fig loads the configuration.
type Option func(f *fig)

// File returns an option that configures the filename that fig
// looks for to provide the config values.
//
// The name must include the extension of the file. Supported
// file types are `yaml`, `yml`, `json` and `toml`.
//
//   fig.Load(&cfg, fig.File("config.toml"))
//
// If this option is not used then fig looks for a file with name `config.yaml`.
func File(name string) Option {
	return func(f *fig) {
		f.filename = name
	}
}

// Dirs returns an option that configures the directories that fig searches
// to find the configuration file.
//
// Directories are searched sequentially and the first one with a matching config file is used.
//
// This is useful when you don't know where exactly your configuration will be during run-time:
//
//   fig.Load(&cfg, fig.Dirs(".", "/etc/myapp", "/home/user/myapp"))
//
//
// If this option is not used then fig looks in the directory it is run from.
func Dirs(dirs ...string) Option {
	return func(f *fig) {
		f.dirs = dirs
	}
}

// Tag returns an option that configures the tag key that fig uses
// when for the alt name struct tag key in fields.
//
//  fig.Load(&cfg, fig.Tag("config"))
//
// If this option is not used then fig uses the tag `fig`.
func Tag(tag string) Option {
	return func(f *fig) {
		f.tag = tag
	}
}

// TimeLayout returns an option that conmfigures the time layout that fig uses when
// parsing a time in a config file or in the default tag for time.Time fields.
//
//   fig.Load(&cfg, fig.TimeLayout("2006-01-02"))
//
// If this option is not used then fig parses times using `time.RFC3339` layout.
func TimeLayout(layout string) Option {
	return func(f *fig) {
		f.timeLayout = layout
	}
}

// UseEnv returns an option that configures fig to additionally load values
// from the environment, after it has loaded values from a config file.
//
//   fig.Load(&cfg, fig.UseEnv("my_app"))
//
// This is meant to be used in conjunction with loading from a file. There
// is no support to ONLY load from the environment.
//
// Fig looks for environment variables in the format PREFIX_FIELD_PATH or
// FIELD_PATH if prefix is empty. Prefix is capitalised regardless of what
// is provided. The field's path is formed by prepending its name with the
// names of all surrounding fields up to the root struct. If a field has
// an alternative name defined inside a struct tag then that name is
// preferred.
//
//   type Config struct {
//     Build    time.Time
//     LogLevel string `fig:"log_level"`
//     Server   struct {
//       Host string
//     }
//   }
//
// With the struct above and UseEnv("myapp") fig would search for the following
// environment variables:
//
//   MYAPP_BUILD
//   MYAPP_LOG_LEVEL
//   MYAPP_SERVER_HOST
func UseEnv(prefix string) Option {
	return func(f *fig) {
		f.useEnv = true
		f.envPrefix = prefix
	}
}
