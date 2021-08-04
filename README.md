<p align="center">
    <img src="img/fig.logo.png" alt="fig" title="fig" class="img-responsive" />
</p>

<p align="center">
    <a href="https://pkg.go.dev/github.com/kkyr/fig?tab=doc"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white" alt="godoc" title="godoc"/></a>
    <a href="https://github.com/kkyr/fig/releases"><img src="https://img.shields.io/github/v/tag/kkyr/fig" alt="semver tag" title="semver tag"/></a>
    <a href="https://goreportcard.com/report/github.com/kkyr/fig"><img src="https://goreportcard.com/badge/github.com/kkyr/fig" alt="go report card" title="go report card"/></a>
    <a href="https://coveralls.io/github/kkyr/fig?branch=master"><img src="https://coveralls.io/repos/github/kkyr/fig/badge.svg?branch=master" alt="coverage status" title="coverage status"/></a>
    <a href="https://github.com/kkyr/fig/blob/master/LICENSE"><img src="https://img.shields.io/github/license/kkyr/fig" alt="license" title="license"/></a>
</p>

# fig

fig is a tiny library for loading an application's config file and its environment into a Go struct. Individual fields can have default values defined or be marked as required.

## Why fig?

- Define your **configuration**, **validations** and **defaults** in a single location
- Optionally **load from the environment** as well
- Only **3** external dependencies
- Full support for`time.Time`, `time.Duration` & `regexp.Regexp`
- Tiny API
- Decoders for `.yaml`, `.json` and `.toml` files

## Getting Started

`$ go get -d github.com/kkyr/fig`

Define your config file:

```yaml
# config.yaml

build: "2020-01-09T12:30:00Z"

server:
    ports:
      - 8080
    cleanup: 1h

logger:
    level: "warn"
    trace: true
```

Define your struct along with _validations_ or _defaults_:

```go
package main

import (
  "fmt"

  "github.com/kkyr/fig"
)

type Config struct {
  Build  time.Time `fig:"build" validate:"required"`
  Server struct {
    Host    string        `fig:"host" default:"127.0.0.1"`
    Ports   []int         `fig:"ports" default:"[80,443]"`
    Cleanup time.Duration `fig:"cleanup" default:"30m"`
  }
  Logger struct {
    Level   string         `fig:"level" default:"info"`
    Pattern *regexp.Regexp `fig:"pattern" default:".*"`
    Trace   bool           `fig:"trace"`
  }
}

func main() {
  var cfg Config
  err := fig.Load(&cfg)
  // handle your err
  
  fmt.Printf("%+v\n", cfg)
  // Output: {Build:2019-12-25 00:00:00 +0000 UTC Server:{Host:127.0.0.1 Ports:[8080] Cleanup:1h0m0s} Logger:{Level:warn Pattern:.* Trace:true}}
}
```

If a field is not set and is marked as *required* then an error is returned. If a *default* value is defined instead then that value is used to populate the field.

Fig searches for a file named `config.yaml` in the directory it is run from. Change the lookup behaviour by passing additional parameters to `Load()`:

```go
fig.Load(&cfg,
  fig.File("settings.json"),
  fig.Dirs(".", "/etc/myapp", "/home/user/myapp"),
) // searches for ./settings.json, /etc/myapp/settings.json, /home/user/myapp/settings.json

```

## Environment

Need to additionally fill fields from the environment? It's as simple as:

```go
fig.Load(&cfg, fig.UseEnv("MYAPP"))
```

## Usage

See usage [examples](/examples).

## Documentation

See [go.dev](https://pkg.go.dev/github.com/kkyr/fig?tab=doc) for detailed documentation.

## Contributing

PRs are welcome! Please explain your motivation for the change in your PR and ensure your change is properly tested and documented.
