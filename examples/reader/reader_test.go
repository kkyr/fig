package reader

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/kkyr/fig"
)

type Config struct {
	Database struct {
		Host     string `fig:"host" validate:"required"`
		Port     int    `fig:"port"`
		Name     string `fig:"name" validate:"required"`
		Username string `fig:"username"`
		Password string `fig:"password"`
	}
	Kafka struct {
		Host []string `fig:"host" validate:"required"`
	}
}

//go:embed reference.yaml
var config string

func ExampleLoad() {

	var cfg Config
	if err := fig.Load(&cfg, fig.Reader(strings.NewReader(config), fig.DecoderYaml)); err == nil {
		fmt.Printf("%+v", cfg)
	} else {
		fmt.Print(err)
	}

	// Output:
	// {Database:{Host:db.prod.example.com Port:5432 Name:orders Username:admin Password:S3cr3t-P455w0rd} Kafka:{Host:[kafka1.prod.example.com kafka2.prod.example.com]}}
}
