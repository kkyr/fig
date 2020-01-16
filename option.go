/*
Copyright 2020 Kyriacos Kyriacou <contact@kkyr.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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

// Tag returns an option that configures the tag that fig uses
// when searching for struct tags in fields.
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
