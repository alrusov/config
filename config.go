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
		LogBufferDelay  Duration       `toml:"log-buffer-delay"`
		LogMaxStringLen int            `toml:"log-max-string-len"`

		GoMaxProcs int `toml:"go-max-procs"`
		GCPercent  int `toml:"gc-percent"`

		MemStatsPeriod Duration `toml:"mem-stats-period"`
		MemStatsLevel  string   `toml:"mem-stats-level"`

		LoadAvgPeriod Duration `toml:"load-avg-period"`

		ProfilerEnabled bool `toml:"profiler-enabled"`
		DeepProfiling   bool `toml:"deep-profiling"`

		UseStdJSON bool `toml:"use-std-json"`

		// Default values for stdhttp callers
		SkipTLSVerification bool `toml:"skip-tls-verification"`
		MinSizeForGzip      int  `toml:"min-size-for-gzip"`

		MaxWorkersCount int `toml:"max-workers-count"`
	}

	// Listener --
	Listener struct {
		// Addr should be set to the desired listening host:port
		Addr      string `toml:"bind-addr"`
		DebugAddr string `toml:"debug-bind-addr"`

		Root string `toml:"root"` // in filesystem

		ProxyPrefix string `toml:"proxy-prefix"`

		// Set certificate in order to handle HTTPS requests
		SSLCombinedPem string `toml:"ssl-combined-pem"`

		//
		Timeout Duration `toml:"timeout"`

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

		Realm string `toml:"realm"`

		Methods             map[string]*AuthMethod `toml:"methods"`
		LocalAdminGroups    []string               `toml:"local-auth-groups"`
		LocalAdminGroupsMap misc.BoolMap           `toml:"-"`
	}

	// User --
	User struct {
		Password string
		Groups   []string
	}

	// AuthMethod --
	AuthMethod struct {
		Enabled bool `toml:"enabled"`
		Score   int  `toml:"score"`
		Options any  `toml:"options"`
	}

	// DB --
	DB struct {
		Type          string `toml:"type"`
		DSN           string `toml:"dsn"`
		MaxConnection int    `toml:"max-conn"`
		Retry         int    `toml:"retry"`
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
	ListenerDefaultTimeout = Duration(5 * time.Second)

	// ClientDefaultTimeout --
	ClientDefaultTimeout = Duration(5 * time.Second)

	// JWTdefaultLifetimeAccess --
	JWTdefaultLifetimeAccess = Duration(time.Hour)

	// JWTdefaultLifetimeRefresh --
	JWTdefaultLifetimeRefresh = Duration(2 * 24 * time.Hour)
)

//----------------------------------------------------------------------------------------------------------------------------//
