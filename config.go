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

var (
	configText = []byte{}
	fEnv       = os.Environ
)

//----------------------------------------------------------------------------------------------------------------------------//

var (
	rePreprocessor = regexp.MustCompile(`(?:\{)([\$#])([^\}]+)(?:\})`)
	reMultiLine    = regexp.MustCompile(`(?m)[[:space:]]*\\\r?\n[[:space:]]*`)
)

//----------------------------------------------------------------------------------------------------------------------------//

func readFile(name string) ([]byte, error) {
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
	rSize, err := f.Read(data)
	if err != nil {
		return nil, err
	}
	if rSize != rSize {
		return nil, fmt.Errorf("File read error - got %d bytes, expected %d", rSize, fSize)
	}

	data = reMultiLine.ReplaceAll(data, []byte(" "))

	return data, nil
}

//----------------------------------------------------------------------------------------------------------------------------//

func populate(data []byte) (*bytes.Buffer, error) {
	var msgs []string

	data = bytes.TrimSpace(reMultiLine.ReplaceAll(data, []byte(" ")))

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
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	size := stat.Size()

	data := make([]byte, size)
	n, err := f.Read(data)
	if err != nil {
		return err
	}
	if int64(n) != size {
		return fmt.Errorf(`File "%s" was not fully read - expect %d, read %d`, fileName, size, n)
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

	return nil
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetText -- get prepared configuration text
func GetText() []byte {
	return configText
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Common) Check(cfg interface{}) (err error) {
	var msgs []string

	if x.Name == "" {
		x.Name = misc.AppName()
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

		if v.Elem().Kind() != reflect.Struct {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" is not pointer to struct`, x))
			continue
		}

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
