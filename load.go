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

var (
	configText   = []byte{}
	fullConfig   = interface{}(nil)
	commonConfig = (*Common)(nil)

	rePreprocessor = regexp.MustCompile(`(?:\{)([\$#])([^\}]+)(?:\})`)
	reMultiLine    = regexp.MustCompile(`(?m)[[:space:]]*\\\r?\n[[:space:]]*`)

	fEnv = os.Environ
	env  = make(map[string][]byte)
)

//----------------------------------------------------------------------------------------------------------------------------//

func loadEnv() {
	osEnv := fEnv()
	for _, s := range osEnv {
		df := strings.SplitN(s, "=", 2)
		v := ""
		if len(df) > 1 {
			v = df[1]
		}
		env[df[0]] = []byte(v)
	}
}

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
	if fSize == 0 {
		return nil, nil
	}

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
					line = bytes.Replace(line, matches[0], v, -1)
				case "#":
					s := string(matches[2])
					if strings.HasPrefix(s, "include ") {
						p := strings.Split(s, " ")
						if len(p) != 2 {
							msgs = append(msgs, fmt.Sprintf(`Illegal preprocessor command "%s" in line %d`, string(matches[2]), n))
							continue
						}

						var err error
						repl, err := readFile(p[1])
						if err != nil {
							msgs = append(msgs, fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), n))
							continue
						}

						b, err := populate(repl)
						if err != nil {
							msgs = append(msgs, fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), n))
							continue
						}
						defer bufpool.PutBuf(b)

						line = bytes.Replace(line, matches[0], bytes.TrimSpace(b.Bytes()), -1)
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
func LoadFile(fileName string, cfg interface{}) (err error) {
	if len(env) == 0 {
		loadEnv()
	}

	data, err := readFile(fileName)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			msg := new(bytes.Buffer)
			msg.WriteString(err.Error() + "\n>>>\n")
			lines := bytes.Split(data, []byte("\n"))
			for i, line := range lines {
				msg.WriteString(fmt.Sprintf("%04d | %s\n", i+1, bytes.TrimSpace(line)))
			}
			msg.WriteString("<<<")
			err = fmt.Errorf(string(msg.Bytes()))
		}
	}()

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
