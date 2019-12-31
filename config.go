package config

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/naoina/toml"

	"github.com/alrusov/bufpool"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Common --
type Common struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Class       string `toml:"class"`

	LogLocalTime   bool   `toml:"log-local-time"`
	LogDir         string `toml:"log-dir"`
	LogLevel       string `toml:"log-level"`
	LogBufferSize  int    `toml:"log-buffer-size"`
	LogBufferDelay int    `toml:"log-buffer-delay"`

	GoMaxProcs int `toml:"go-max-procs"`

	MemStatsPeriod int    `toml:"mem-stats-period"`
	MemStatsLevel  string `toml:"mem-stats-level"`

	LoadAvgPeriod int `toml:"load-avg-period"`

	ProfilerEnabled bool `toml:"profiler-enabled"`
	DeepProfiling   bool `toml:"deep-profiling"`

	DisabledEndpointsSlice []string        `toml:"disabled-endpoints"`
	DisabledEndpoints      map[string]bool `toml:"-"`
}

// Listener --
type Listener struct {
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

var (
	configText   = []byte{}
	fullConfig   = interface{}(nil)
	commonConfig = (*Common)(nil)
	fEnv         = os.Environ
)

//----------------------------------------------------------------------------------------------------------------------------//

var (
	rePreprocessor = regexp.MustCompile(`(?:\{)([\$#])([^\}]+)(?:\})`)
	reMultiLine    = regexp.MustCompile(`(?m)[[:space:]]*\\\r?\n[[:space:]]*`)
)

//----------------------------------------------------------------------------------------------------------------------------//

func readFile(name string) ([]byte, error) {
	name, err := misc.AbsPath(name)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	fSize := fi.Size()
	data := make([]byte, fSize)
	dSize, err := f.Read(data)
	if err != nil {
		return nil, err
	}
	if int64(dSize) != fSize {
		return nil, fmt.Errorf("File read error - got %d bytes, expected %d", dSize, fSize)
	}

	data = bytes.TrimSpace(reMultiLine.ReplaceAll(data, []byte(" ")))

	return data, nil
}

//----------------------------------------------------------------------------------------------------------------------------//

func populate(data []byte) (*bytes.Buffer, error) {
	var msgs []string

	osEnv := fEnv()
	env := make(map[string][]byte, len(osEnv))
	for _, s := range osEnv {
		df := strings.SplitN(s, "=", 2)
		v := ""
		if len(df) > 1 {
			v = df[1]
		}
		env[df[0]] = []byte(v)
	}

	newData := bufpool.GetBuf()
	list := bytes.Split(data, []byte("\n"))
	n := 0

	for _, line := range list {
		n++
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}

		findResult := rePreprocessor.FindAllSubmatch(line, -1)
		if findResult != nil {
			for _, matches := range findResult {
				switch string(matches[1]) {
				case "$":
					name := string(matches[2])
					v, exists := env[name]
					if !exists {
						msgs = append(msgs, fmt.Sprintf(`Undefined environment variable "%s" in line %d`, name, n))
						continue
					}
					line = bytes.Replace(line, []byte("{$"+name+"}"), v, -1)
				case "#":
					s := string(matches[2])
					if strings.HasPrefix(s, "include ") {
						p := strings.Split(s, " ")
						if len(p) != 2 {
							msgs = append(msgs, fmt.Sprintf(`Illegal preprocessor command "%s" in line %d`, string(matches[2]), n))
							continue
						}

						var err error
						line, err = readFile(p[1])
						if err != nil {
							msgs = append(msgs, fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), n))
							continue
						}

						b, err := populate(line)
						if err != nil {
							msgs = append(msgs, fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), n))
							continue
						}
						defer bufpool.PutBuf(b)

						line = b.Bytes()
						continue
					}

					msgs = append(msgs, fmt.Sprintf(`Unknown preprocessor command "%s" in line %d`, string(matches[2]), n))
				}
			}
		}

		if len(line) != 0 {
			newData.Write(bytes.TrimSpace(line))
			newData.WriteByte(byte('\n'))
		}
	}

	if len(msgs) != 0 {
		bufpool.PutBuf(newData)
		return nil, misc.JoinedError(msgs)
	}

	return newData, nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// LoadFile parses the specified file into a Config object
func LoadFile(fileName string, cfg interface{}) error {
	data, err := readFile(fileName)
	if err != nil {
		return err
	}

	newData, err := populate(data)
	if err != nil {
		return err
	}
	defer bufpool.PutBuf(newData)

	data = newData.Bytes()
	configText = make([]byte, len(data))
	copy(configText, data)

	err = toml.Unmarshal(newData.Bytes(), cfg)
	if err != nil {
		return err
	}

	fullConfig = cfg

	cp := reflect.ValueOf(cfg)
	if cp.Kind() == reflect.Ptr {
		c := cp.Elem()
		if c.Kind() == reflect.Struct {
			fCnt := c.NumField()
			for i := 0; i < fCnt; i++ {
				f, ok := c.Field(i).Addr().Interface().(*Common)
				if ok {
					commonConfig = f
					break
				}
			}
		}
	}

	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetText -- get prepared configuration text
func GetText() []byte {
	return configText
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetConfig --
func GetConfig() interface{} {
	return fullConfig
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetCommon --
func GetCommon() *Common {
	return commonConfig
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Common) Check(cfg interface{}) (err error) {
	var msgs []string

	if x.Name == "" {
		x.Name = misc.AppName()
	}

	if x.LoadAvgPeriod <= 0 {
		x.LoadAvgPeriod = 60
	}

	x.DisabledEndpoints = make(map[string]bool, len(x.DisabledEndpointsSlice))
	for _, name := range x.DisabledEndpointsSlice {
		name = misc.NormalizeSlashes(name)
		x.DisabledEndpoints[name] = true
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func Check(cfg interface{}, list []interface{}) error {
	var msgs []string

	for _, x := range list {
		v := reflect.ValueOf(x)

		if v.Kind() != reflect.Ptr {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" is not pointer`, x))
			continue
		}

		//if v.Elem().Kind() != reflect.Struct {
		//	misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" is not pointer to struct`, x))
		//	continue
		//}

		m := v.MethodByName("Check")

		if m.Kind() != reflect.Func {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" doesn't have the "Check" method`, x))
			continue
		}

		e := m.Call([]reflect.Value{reflect.ValueOf(cfg)})

		if len(e) != 1 || e[0].Kind() != reflect.Interface {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" method "Check" returned illegal value`, x))
			continue
		}

		if e[0].IsNil() {
			continue
		}

		err, ok := e[0].Interface().(error)
		if !ok {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" method "Check" returned not error value`, x))
			continue
		}

		if err == nil {
			continue
		}

		misc.AddMessage(&msgs, err.Error())
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//
