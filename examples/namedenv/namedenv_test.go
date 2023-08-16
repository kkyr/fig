package namedenv

import (
	"fmt"
	"os"

	"github.com/kkyr/fig"
)

type Config struct {
	Winrm struct {
		User string `validate:"required" fig:"user"`
		Pwd  string `validate:"required" fig:"pwd"`
	} `fig:"winrm"`

	LockMsg string `fig:"lock_msg"`
}

func ExampleLoad() {
	os.Clearenv()
	check(os.Setenv("CI_CONNECT_PWD", "securepassword"))

	var cfg Config
	err := fig.Load(&cfg, fig.UseNamedEnv())
	check(err)

	fmt.Println(cfg.Winrm.User)
	fmt.Println(cfg.Winrm.Pwd)
	fmt.Println(cfg.LockMsg)

	// Output:
	// ci-robot
	// securepassword
	// is locked by CI
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
