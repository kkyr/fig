package custom

import (
	"fmt"
	"strings"

	"github.com/kkyr/fig"
)

type ListenerType uint

const (
	ListenerUnix ListenerType = iota
	ListenerTCP
	ListenerTLS
)

type Config struct {
	App struct {
		Environment string `fig:"environment" validate:"required"`
	} `fig:"app"`
	Server struct {
		Host     string       `fig:"host" default:"0.0.0.0"`
		Port     int          `fig:"port" default:"80"`
		Listener ListenerType `fig:"listener_type" default:"tcp"`
	} `fig:"server"`
}

func ExampleLoad() {
	var cfg Config
	err := fig.Load(&cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg.App.Environment)
	fmt.Println(cfg.Server.Host)
	fmt.Println(cfg.Server.Port)
	fmt.Println(cfg.Server.Listener)

	// Output:
	// dev
	// 0.0.0.0
	// 443
	// tcp
}

func (l *ListenerType) UnmarshalString(v string) error {
	switch strings.ToLower(v) {
	case "unix":
		*l = ListenerUnix
	case "tcp":
		*l = ListenerTCP
	case "tls":
		*l = ListenerTLS
	default:
		return fmt.Errorf("unknown listener type: %s", v)
	}
	return nil
}

func (l ListenerType) String() string {
	switch l {
	case ListenerUnix:
		return "unix"
	case ListenerTCP:
		return "tcp"
	case ListenerTLS:
		return "tls"
	default:
		return "unknown"
	}
}
