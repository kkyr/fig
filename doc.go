/*
Package fig loads configuration files into Go structs with extra juice for validating fields and setting defaults.

Config files may be defined in in yaml, json or toml format.

Example

Define your configuration file in the root of your project:

  # config.yaml

  build: "2020-01-09T12:30:00Z"

  server:
    ports:
      - 8080
    cleanup: 1h

  logger:
    level: "warn"
    trace: true

Define your struct and load it:

 package main

 import (
   "fmt"

   "github.com/kkyr/fig"
 )


  type Config struct {
    Build  time.Time `fig:"build,required"`
    Server struct {
      Host    string        `fig:"host,default=127.0.0.1"`
      Ports   []int         `fig:"ports,default=[80,443]"`
      Cleanup time.Duration `fig:"cleanup,default=30m"`
    }
    Logger struct {
      Level string `fig:"level,default=info"`
      Trace bool   `fig:"trace"`
    }
  }

 func main() {
   var cfg Config
   _ = fig.Load(&cfg)

   fmt.Printf("%+v\n", cfg)
   // Output: {Build:2019-12-25 00:00:00 +0000 UTC Server:{Host:127.0.0.1 Ports:[8080] Cleanup:1h0m0s} Logger:{Level:warn Trace:true}}
 }

By default fig searches for a file named `config.yaml` in the directory it is run from.
It can be configured to look elsewhere.

Configuration

Pass options as additional parameters to `Load()` to configure fig's behaviour.

File

Change the file and directories fig searches in with `File()`.

  fig.Load(&cfg,
    fig.File("settings.json"),
    fig.Dirs(".", "home/user/myapp", "/opt/myapp"),
  )

Fig searches for the file in dirs sequentially and uses the first matching file.

The decoder (yaml/json/toml) used is picked based on the file's extension.

Tag

The name of the struct tag that fig uses can be changed with `Tag()`.

  type Config struct {
    Host  string `config:"host,required"`
    Level string `config:"level,default=info"`
  }

  var cfg Config
  fig.Load(&cfg, fig.Tag("config"))

By default fig uses the tag name `fig`.

Time

Change the layout fig uses to parse times using `TimeLayout()`.

  type Config struct {
    Date time.Time `fig:"date,default=12-25-2019"`
  }

  var cfg Config
  fig.Load(&cfg, fig.TimeLayout("01-02-2006"))

  fmt.Printf("%+v", cfg)
  // Output: {Date:2019-12-25 00:00:00 +0000 UTC}

By default fig parses time using the `RFC.3339` layout (`2006-01-02T15:04:05Z07:00`).

Validation

Fields can be validated by adding an appropriate key to the field tag.
A maximum of one validation may be added to each field.

Required

A required key in the field tag causes fig to check if the field has been set after it's loaded from the config file. Required fields that are not set are returned as an error.

  type Config struct {
    Host string `fig:"host,required"` // or `fig:",required"
  }

Fig uses the following properties to check if a field is set:

  basic types:           != to its zero value ("" for str, 0 for int, etc.)
  slices, arrays:        len() > 0
  pointers*, interfaces: != nil
  structs:               always true (use a struct pointer to check for struct presence)
  time.Time:             !time.IsZero()
  time.Duration:         != 0

  *non-nil pointers to non-struct types (except time.Time) are de-referenced and then checked

See example below to help understand:

  type Config struct {
    A string    `fig:",required"`
    B *string   `fig:",required"`
    C int       `fig:",required"`
    D *int      `fig:",required"`
    E []float32 `fig:",required"`
    F struct{}  `fig:",required"`
    G *struct{} `fig:",required"`
    H struct {
      I interface{} `fig:",required"`
      J interface{} `fig:",required"`
    } `fig:",required"`
    K *[]bool    `fig:",required"`
    L []uint     `fig:",required"`
    M *time.Time `fig:",required"`
  }

  var cfg Config

  // simulate loading of config file
  b := ""
  cfg.B = &b
  cfg.H.I = 5.5
  cfg.K = &[]bool{}
  cfg.L = []uint{5}
  m := time.Time{}
  cfg.M = &m

  err := fig.Load(&cfg)
  fmt.Print(err)
  // A: required, B: required, C: required, D: required, E: required, G: required, H.J: required, K: required, M: required

Default

A default key in the field tag causes fig to fill the field with the value specified only if the field is not set.
It must be in the format default=value.

Fig attempts to parse the value based on the field's type. If parsing fails then an error is returned.

  type Config struct {
    Port int `fig:"port,default=8000"` // or `fig:",default=8000"
  }


A default value can be set for the following types:

  all basic types except bool and complex
  time.Time
  time.Duration
  slices (of above types)

Slice defaults must be enclosed in square brackets and successive values separated by a comma:

  type Config struct {
    Durations []time.Duration `fig:",default=[30m,1h,90m,2h]"
  }

Note: the default setter knows if it should fill a field or not by comparing if the current value of the field is equal to the corresponding zero value for that field's type. This happens after the configuration is loaded and has the implication that the zero value set explicitly by the user will get overwritten by any default value registered for that field. It's for this reason that defaults on booleans are not permitted, as a boolean field with a default value of `true` would always be true (since if it were set to false it'd be overwritten).

Errors

A wrapped error `ErrFileNotFound` is returned when fig is not able to find a config file to load. This can be useful for instance to fallback to a different configuration loading mechanism.

  var cfg Config
  err := fig.Load(&cfg)
  if errors.Is(err, fig.ErrFileNotFound) {
    // load config from elsewhere
  }
*/
package fig
