package profile

import (
	"fmt"
	"os"

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

func ExampleLoad() {
	var cfg Config
	if err := fig.Load(&cfg); err == nil {
		fmt.Printf("%+v", cfg)
	}

	// Output:
	// {Database:{Host:db.prod.example.com Port:5432 Name:users Username:admin Password:S3cr3t-P455w0rd} Kafka:{Host:[kafka1.prod.example.com kafka2.prod.example.com]}}
}

func ExampleLoad_with_environment_in_config_file() {
	os.Setenv("DATABASE_NAME", "users-readonly")

	var cfg Config
	if err := fig.Load(&cfg); err == nil {
		fmt.Printf("%+v", cfg)
	}

	// Output:
	// {Database:{Host:db.prod.example.com Port:5432 Name:users-readonly Username:admin Password:S3cr3t-P455w0rd} Kafka:{Host:[kafka1.prod.example.com kafka2.prod.example.com]}}
}

func ExampleLoad_with_multi_profile() {
	var cfg Config
	if err := fig.Load(&cfg, fig.Profiles("test", "integration"), fig.ProfileLayout("config-test.yaml")); err == nil {
		fmt.Printf("%+v", cfg)
	}

	// Output:
	// {Database:{Host:db Port:5432 Name:users Username:admin Password:postgres} Kafka:{Host:[kafka]}}
}

func ExampleLoad_with_single_profile() {
	var cfg Config
	if err := fig.Load(&cfg, fig.Profiles("test"), fig.ProfileLayout("config-test.yaml")); err == nil {
		fmt.Printf("%+v", cfg)
	}

	// Output:
	// {Database:{Host:sqlite:file.db Port:-1 Name:users Username: Password:} Kafka:{Host:[embedded:kafka]}}
}
