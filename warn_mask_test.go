package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Дамп конфига при предупреждении загрузки (неопределённая переменная окружения) не выносит секреты в лог
func TestWarningDumpIsSecured(t *testing.T) {
	const (
		password = "IdcTestPassw0rd"
		secret   = "IdcTestJwtS3cret"
	)

	isolateEnv(t)

	dir := t.TempDir()
	fn := filepath.Join(dir, "warn.toml")
	src := `x = "${IDC_CONFIG_TEST_UNDEFINED_VAR}"
password = "` + password + `"
jwt = { secret = "` + secret + `" }
`
	if err := os.WriteFile(fn, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	log.SetConsoleWriter(buf)
	defer log.SetConsoleWriter(os.Stderr)

	cfg := struct {
		X        string            `toml:"x"`
		Password string            `toml:"password"`
		Jwt      map[string]string `toml:"jwt"`
	}{}

	if err := LoadFile(fn, &cfg); err != nil {
		t.Fatal(err)
	}

	out := buf.String()

	if !strings.Contains(out, "Config file:") {
		t.Fatalf("no config dump in the log, the test does not reach the dump path:\n%s", out)
	}

	for _, v := range []string{password, secret} {
		if strings.Contains(out, v) {
			t.Errorf("config dump contains a plain secret value (%d chars)", len(v))
		}
	}

	if cfg.Password != password || cfg.Jwt["secret"] != secret {
		t.Errorf("masking must not touch loaded values")
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
