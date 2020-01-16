<p align="center">
    <img src="img/fig.logo.png" alt="fig" title="fig" height="384" />
</p>

<p align="center">
    <a href="https://godoc.org/github.com/kkyr/fig"><img src="https://godoc.org/github.com/kkyr/fig?status.svg" alt="godoc" title="godoc"/></a>
    <a href="https://travis-ci.org/kkyr/fig"><img src="https://travis-ci.org/kkyr/fig.svg?branch=master" alt="build status" title="build status"/></a>
    <a href="https://github.com/kkyr/fig/releases"><img src="https://img.shields.io/github/v/tag/kkyr/fig" alt="semver tag" title="semver tag"/></a>
    <a href="https://goreportcard.com/report/github.com/kkyr/fig"><img src="https://goreportcard.com/badge/github.com/kkyr/fig" alt="go report card" title="go report card"/></a>
    <a href="https://coveralls.io/github/kkyr/fig?branch=master"><img src="https://coveralls.io/repos/github/kkyr/fig/badge.svg?branch=master" alt="coverage status" title="coverage status"/></a>
    <a href="https://github.com/kkyr/fig/blob/master/LICENSE"><img src="https://img.shields.io/github/license/kkyr/fig" alt="license" title="license"/></a>
</p>

# fig

fig loads configuration files into Go structs with extra juice for validating fields and setting defaults.

## Why fig?

Define your config, validations and defaults all within a single struct. Fig does the rest!

Additionally, fig:

- Understands `time.Time` & `time.Duration`
- Has only 3 external dependencies
- Exposes a tiny API
- Decodes `.yaml`, `.json` and `.toml` files
- Is extensively tested

## Getting Started

`$ go get -d github.com/kkyr/fig`

Define your configuration file:

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

Define your struct and load it using fig:

```go
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
  err := fig.Load(&cfg)
  // handle your err
  
  fmt.Printf("%+v\n", cfg)
  // Output: {Build:2019-12-25 00:00:00 +0000 UTC Server:{Host:127.0.0.1 Ports:[8080] Cleanup:1h0m0s} Logger:{Level:warn Trace:true}}
}
```

Fig searches by default for a file named `config.yaml` in the directory it is run from.

Change the behaviour based on your needs by passing additional parameters to `Load()`:

```go
fig.Load(&cfg,
  fig.File("settings.json"),
  fig.Dirs(".", "/etc/myapp", "/home/user/myapp"),
) // searches for ./settings.json, /etc/myapp/settings.json, /home/user/myapp/settings.json

```

## Usage

See [godoc](https://godoc.org/github.com/kkyr/fig) for detailed usage documentation.

## Contributing

PRs are welcome! Please ensure you add relevant tests & documentation prior to making one.
