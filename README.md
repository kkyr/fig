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

fig is a tiny library for loading an application's configuration into a Go struct.

## Why fig?

- 🛠️ Define your **configuration**, **validations** and **defaults** all within a single struct.
- 🌍 Easily load your configuration from a **file**, the **environment**, or both.
- ⏰ Decode strings into `Time`, `Duration`, `Regexp`, or any custom type that satisfies the `StringUnmarshaler` interface.
- 🗂️ Compatible with `yaml`, `json`, and `toml` file formats.
- 🧩 Only three external dependencies.

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
  // error handling omitted
  
  fmt.Printf("%+v\n", cfg)
  // {Build:2019-12-25T00:00:00Z Server:{Host:127.0.0.1 Ports:[8080] Cleanup:1h0m0s} Logger:{Level:warn Pattern:.* Trace:true}}
}
```

Fields marked as _required_ are checked to ensure they're not empty, and _default_ values are applied to fill in those that are empty.
```

## Environment

By default, fig will only look for values in a config file. To also include values from the environment, use the `UseEnv` option:

```go
fig.Load(&cfg, fig.UseEnv("APP_PREFIX"))
```

In case of conflicts, values from the environment take precedence.

## Usage

See usage [examples](/examples).

## Documentation

For detailed documentation, visit [go.dev](https://pkg.go.dev/github.com/kkyr/fig?tab=doc).

## Contributing

PRs are welcome! Please explain your motivation for the change in your PR and ensure your change is properly tested and documented.
