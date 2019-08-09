package config

import (
	"os"

	"github.com/naoina/toml"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Common --
type Common struct {
	LogLocalTime   bool   `toml:"log-local-time"`
	LogDir         string `toml:"log-dir"`
	LogLevel       string `toml:"log-level"`
	LogBufferSize  int    `toml:"log-buffer-size"`
	LogBufferDelay int    `toml:"log-buffer-delay"`

	GoMaxProcs int `toml:"go-max-procs"`

	MemStatsPeriod int    `toml:"mem-stats-period"`
	MemStatsLevel  string `toml:"mem-stats-level"`
}

// Listener --
type Listener struct {
	// Addr should be set to the desired listening host:port
	Addr string `toml:"bind-addr"`

	// Set certificate in order to handle HTTPS requests
	SSLCombinedPem string `toml:"ssl-combined-pem"`

	//
	Timeout int `toml:"timeout"`
}

// DB --
type DB struct {
	Type    string `toml:"type"`
	DSN     string `toml:"dsn"`
	Timeout int    `toml:"timeout"`
	Retry   int    `toml:"retry"`
}

const (
	// ListenerDefaultTimeout --
	ListenerDefaultTimeout = 5

	// ClientDefaultTimeout --
	ClientDefaultTimeout = 5
)

//----------------------------------------------------------------------------------------------------------------------------//

// LoadFile parses the specified file into a Config object
func LoadFile(filename string, cfg interface{}) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewDecoder(f).Decode(cfg)
}

//----------------------------------------------------------------------------------------------------------------------------//
