package config

import (
	"time"

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
		LogBufferDelayS string         `toml:"log-buffer-delay"`
		LogBufferDelay  time.Duration  `toml:"-"`
		LogMaxStringLen int            `toml:"log-max-string-len"`

		GoMaxProcs int `toml:"go-max-procs"`

		MemStatsPeriodS string        `toml:"mem-stats-period"`
		MemStatsPeriod  time.Duration `toml:"-"`
		MemStatsLevel   string        `toml:"mem-stats-level"`

		LoadAvgPeriodS string        `toml:"load-avg-period"`
		LoadAvgPeriod  time.Duration `toml:"-"`

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
		TimeoutS string        `toml:"timeout"`
		Timeout  time.Duration `toml:"-"`

		IconFile string `toml:"icon-file"`

		DisabledEndpointsSlice []string     `toml:"disabled-endpoints"`
		DisabledEndpoints      misc.BoolMap `toml:"-"`

		Auth Auth `toml:"auth"`
	}

	// Auth --
	Auth struct {
		EndpointsSlice map[string][]string     `toml:"endpoints"`
		Endpoints      map[string]misc.BoolMap `toml:"-"`

		UsersMap misc.StringMap  `toml:"users"`
		Users    map[string]User `toml:"-"`

		Methods map[string]*AuthMethod `toml:"methods"`
	}

	// User --
	User struct {
		Password string
		Groups   []string
	}

	// AuthMethod --
	AuthMethod struct {
		Enabled    bool              `toml:"enabled"`
		Score      int               `toml:"score"`
		OptionsMap misc.InterfaceMap `toml:"options"`
		Options    interface{}       `toml:"-"`
	}

	// DB --
	DB struct {
		Type          string        `toml:"type"`
		DSN           string        `toml:"dsn"`
		MaxConnection int           `toml:"max-conn"`
		Retry         int           `toml:"retry"`
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

const (
	// AuthMethodBasic --
	AuthMethodBasic = "basic"
	// AuthMethodJWT --
	AuthMethodJWT = "jwt"
	// AuthMethodKrb5 --
	AuthMethodKrb5 = "krb5"
)

const (
	// ListenerDefaultTimeout --
	ListenerDefaultTimeout = 5 * time.Second

	// ClientDefaultTimeout --
	ClientDefaultTimeout = 5 * time.Second

	// JWTdefaultLifetime --
	JWTdefaultLifetime = 3600 * time.Second
)

//----------------------------------------------------------------------------------------------------------------------------//
