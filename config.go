package config

import (
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type (
	// App --
	App interface {
		Check() (err error)
	}

	// Common --
	Common struct {
		Name        string `toml:"name"`
		Description string `toml:"description"`
		Class       string `toml:"class"`

		LogLocalTime    bool           `toml:"log-local-time"`
		LogDir          string         `toml:"log-dir"`
		LogLevel        string         `toml:"log-level"`  // default
		LogLevels       misc.StringMap `toml:"log-levels"` // by facilities
		LogBufferSize   int            `toml:"log-buffer-size"`
		LogBufferDelay  int            `toml:"log-buffer-delay"`
		LogMaxStringLen int            `toml:"log-max-string-len"`

		GoMaxProcs int `toml:"go-max-procs"`

		MemStatsPeriod int    `toml:"mem-stats-period"`
		MemStatsLevel  string `toml:"mem-stats-level"`

		LoadAvgPeriod int `toml:"load-avg-period"`

		ProfilerEnabled bool `toml:"profiler-enabled"`
		DeepProfiling   bool `toml:"deep-profiling"`

		UseStdJSON bool `toml:"use-std-json"`

		MinSizeForGzip int `toml:"min-size-for-gzip"`
	}

	// Listener --
	Listener struct {
		// Addr should be set to the desired listening host:port
		Addr string `toml:"bind-addr"`

		Root string `toml:"root"` // in filesystem

		ProxyPrefix string `toml:"proxy-prefix"`

		// Set certificate in order to handle HTTPS requests
		SSLCombinedPem string `toml:"ssl-combined-pem"`

		//
		Timeout int `toml:"timeout"`

		IconFile string `toml:"icon-file"`

		DisabledEndpointsSlice []string     `toml:"disabled-endpoints"`
		DisabledEndpoints      misc.BoolMap `toml:"-"`

		Auth Auth `toml:"auth"`
	}

	// Auth --
	Auth struct {
		EndpointsSlice map[string][]string     `toml:"endpoints"`
		Endpoints      map[string]misc.BoolMap `toml:"-"`

		Users misc.StringMap `toml:"users"`

		Methods map[string]*AuthMethod `toml:"methods"`
	}

	// AuthMethod --
	AuthMethod struct {
		Enabled    bool              `toml:"enabled"`
		OptionsMap misc.InterfaceMap `toml:"options"`
		Options    interface{}       `toml:"-"`
	}

	// DB --
	DB struct {
		Type    string `toml:"type"`
		DSN     string `toml:"dsn"`
		Timeout int    `toml:"timeout"`
		Retry   int    `toml:"retry"`
	}
)

const (
	// AuthMethodBasic --
	AuthMethodBasic = "basic"
	// AuthMethodJWT --
	AuthMethodJWT = "jwt"
	// AuthMethodKrb5 --
	AuthMethodKrb5 = "krb5"
)

//----------------------------------------------------------------------------------------------------------------------------//

const (
	// ListenerDefaultTimeout --
	ListenerDefaultTimeout = 5

	// ClientDefaultTimeout --
	ClientDefaultTimeout = 5

	// JWTdefaultLifetime --
	JWTdefaultLifetime = 3600
)

//----------------------------------------------------------------------------------------------------------------------------//
