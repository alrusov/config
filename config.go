package config

import (
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// Common --
	Common struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
		Class       string `toml:"class"`

		LogLocalTime    bool   `toml:"log-local-time"`
		LogDir          string `toml:"log-dir"`
		LogLevel        string `toml:"log-level"`
		LogBufferSize   int    `toml:"log-buffer-size"`
		LogBufferDelay  int    `toml:"log-buffer-delay"`
		LogMaxStringLen int    `toml:"log-max-string-len"`

		GoMaxProcs int `toml:"go-max-procs"`

		MemStatsPeriod int    `toml:"mem-stats-period"`
		MemStatsLevel  string `toml:"mem-stats-level"`

		LoadAvgPeriod int `toml:"load-avg-period"`

		ProfilerEnabled bool `toml:"profiler-enabled"`
		DeepProfiling   bool `toml:"deep-profiling"`

		DisabledEndpointsSlice []string        `toml:"disabled-endpoints"`
		DisabledEndpoints      map[string]bool `toml:"-"`

		MinSizeForGzip int `toml:"timeout"`
	}

	// Listener --
	Listener struct {
		// Addr should be set to the desired listening host:port
		Addr string `toml:"bind-addr"`

		// Set certificate in order to handle HTTPS requests
		SSLCombinedPem string `toml:"ssl-combined-pem"`

		//
		Timeout int `toml:"timeout"`

		IconFile string `toml:"icon-file"`

		BasicAuthEnabled bool           `toml:"basic-auth-enabled"`
		Users            misc.StringMap `toml:"users"`
	}

	// DB --
	DB struct {
		Type    string `toml:"type"`
		DSN     string `toml:"dsn"`
		Timeout int    `toml:"timeout"`
		Retry   int    `toml:"retry"`
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

const (
	// ListenerDefaultTimeout --
	ListenerDefaultTimeout = 5

	// ClientDefaultTimeout --
	ClientDefaultTimeout = 5
)

//----------------------------------------------------------------------------------------------------------------------------//
