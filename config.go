package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/naoina/toml"

	"github.com/alrusov/bufpool"
	"github.com/alrusov/misc"
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

var (
	configText = []byte{}
)

//----------------------------------------------------------------------------------------------------------------------------//

var (
	re = regexp.MustCompile(`(?:\{\$)([[:alnum:]_]+)(?:\})`)
)

func populate(data []byte) (*bytes.Buffer, error) {
	var msgs []string

	osEnv := os.Environ()
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
		findResult := re.FindAllSubmatch(line, -1)
		if findResult != nil {
			for _, matches := range findResult {
				name := string(matches[1])
				v, exists := env[name]
				if !exists {
					msgs = append(msgs, fmt.Sprintf(`Undefined environment variable "%s" in line %d`, name, n))
				}
				line = bytes.Replace(line, []byte("{$"+name+"}"), v, -1)
			}
		}
		newData.Write(line)
		newData.WriteByte(byte('\n'))
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

	re := regexp.MustCompile(`(?m)[[:space:]]*\\\r?\n[[:space:]]*`)
	data = re.ReplaceAll(data, []byte(" "))

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
