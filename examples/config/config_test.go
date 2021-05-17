package config

import (
	"fmt"
	"regexp"
	"time"

	"github.com/kkyr/fig"
)

type Config struct {
	App struct {
		Environment string `fig:"environment" validate:"required"`
	} `fig:"app"`
	Server struct {
		Host         string        `fig:"host" default:"0.0.0.0"`
		Port         int           `fig:"port" default:"80"`
		ReadTimeout  time.Duration `fig:"read_timeout" default:"30s"`
		WriteTimeout time.Duration `fig:"write_timeout" default:"30s"`
	} `fig:"server"`
	Logger struct {
		Level   string         `fig:"level" default:"info"`
		Pattern *regexp.Regexp `fig:"pattern" default:".*"`
	} `fig:"logger"`
	Certificate struct {
		Version    int       `fig:"version"`
		DNSNames   []string  `fig:"dns_names" default:"[kkyr,kkyr.io]"`
		Expiration time.Time `fig:"expiration" validate:"required"`
	} `fig:"certificate"`
}

func ExampleLoad() {
	var cfg Config
	err := fig.Load(&cfg, fig.TimeLayout("2006-01-02"))
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.App.Environment)
	fmt.Println(cfg.Server.Host)
	fmt.Println(cfg.Server.Port)
	fmt.Println(cfg.Server.ReadTimeout)
	fmt.Println(cfg.Server.WriteTimeout)
	fmt.Println(cfg.Logger.Level)
	fmt.Println(cfg.Logger.Pattern)
	fmt.Println(cfg.Certificate.Version)
	fmt.Println(cfg.Certificate.DNSNames)
	fmt.Println(cfg.Certificate.Expiration.Format("2006-01-02"))

	// Output:
	// dev
	// 0.0.0.0
	// 443
	// 1m0s
	// 30s
	// debug
	// [a-z]+
	// 1
	// [kkyr kkyr.io]
	// 2020-12-01
}
