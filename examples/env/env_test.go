package env

import (
	"fmt"
	"os"

	"github.com/kkyr/fig"
)

type Config struct {
	Database struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required"`
		Database string `validate:"required" fig:"db"`
		Username string `validate:"required"`
		Password string `validate:"required"`
	}
	Container struct {
		Args []string `default:"[/bin/sh]"`
	}
}

func ExampleLoad() {
	os.Clearenv()
	check(os.Setenv("APP_DATABASE_HOST", "pg.internal.corp"))
	check(os.Setenv("APP_DATABASE_USERNAME", "mickey"))
	check(os.Setenv("APP_DATABASE_PASSWORD", "mouse"))
	check(os.Setenv("APP_CONTAINER_ARGS", "[-p,5050:5050]"))

	var cfg Config
	err := fig.Load(&cfg, fig.UseEnv("app"))
	check(err)

	fmt.Println(cfg.Database.Host)
	fmt.Println(cfg.Database.Port)
	fmt.Println(cfg.Database.Database)
	fmt.Println(cfg.Database.Username)
	fmt.Println(cfg.Database.Password)
	fmt.Println(cfg.Container.Args)

	// Output:
	// pg.internal.corp
	// 5432
	// users
	// mickey
	// mouse
	// [-p 5050:5050]
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
