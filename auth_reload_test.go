package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

//----------------------------------------------------------------------------------------------------------------------------//

type reloadPeriodCfg struct {
	Auth Auth `toml:"auth"`
}

func (c *reloadPeriodCfg) Check(any) error { return nil }

func TestAuthReloadPeriod(t *testing.T) {
	cases := map[string]time.Duration{
		`auth = { reload-period = "90s" }`: 90 * time.Second,
		`auth = { reload-period = "" }`:    0,
		`auth = { }`:                       0,
	}

	for src, exp := range cases {
		fn := filepath.Join(t.TempDir(), "cfg.toml")
		if err := os.WriteFile(fn, []byte(src+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		c := &reloadPeriodCfg{}
		if err := LoadFile(fn, c); err != nil {
			t.Fatalf(`%s: %s`, src, err)
		}

		if got := c.Auth.ReloadPeriod.D(); got != exp {
			t.Errorf(`%s: expected %s, got %s`, src, exp, got)
		}
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
