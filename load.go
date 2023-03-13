package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/naoina/toml"

	"github.com/alrusov/log"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

var (
	configText   = ""
	fullConfig   = any(nil)
	commonConfig = (*Common)(nil)

	rePreprocessor = regexp.MustCompile(`(?:\{)([\$#])([^\}]+)(?:\})`)

	// Use the # symbol at the begining of the line for comment
	reComment = regexp.MustCompile(`(?m)^\s*#.*$`)

	// Use the \ symbol at the end line to continue to next line
	reMultiLine = regexp.MustCompile(`(?m)\s*\\\s*\r?\n\s*`)

	fEnv = os.Environ
	env  = map[string][]byte{}
)

//----------------------------------------------------------------------------------------------------------------------------//

func readFile(name string, base string) ([]byte, string, error) {
	name, err := misc.AbsPathEx(name, base)
	if err != nil {
		return nil, "", err
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, name, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, name, err
	}

	fSize := fi.Size()
	if fSize == 0 {
		return nil, name, nil
	}

	data := make([]byte, fSize)
	dSize, err := f.Read(data)
	if err != nil {
		return nil, name, err
	}
	if int64(dSize) != fSize {
		return nil, name, fmt.Errorf("file read error - got %d bytes, expected %d", dSize, fSize)
	}

	data = bytes.TrimSpace(reComment.ReplaceAll(data, []byte{}))
	data = bytes.TrimSpace(reMultiLine.ReplaceAll(data, []byte{' '}))
	data = bytes.TrimRight(data, "\\")

	return data, name, nil
}

//----------------------------------------------------------------------------------------------------------------------------//

func populate(data []byte, base string, lineNumber *uint) (newData *bytes.Buffer, withWarn bool, err error) {
	n := uint(0)
	if lineNumber == nil {
		lineNumber = &n
	}

	newData = new(bytes.Buffer)
	withWarn = false

	msgs := misc.NewMessages()

	list := bytes.Split(data, []byte("\n"))

	for _, line := range list {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '#' {
			continue
		}

		line = bytes.ReplaceAll(line, []byte("\t"), []byte(" "))

		*lineNumber++

		nIter := 0

		for {
			nIter++
			if nIter > 32 {
				msgs.Add(`Too many iterations for "%s"`, line)
				break
			}

			findResult := rePreprocessor.FindAllSubmatch(line, -1)
			if len(findResult) == 0 {
				break
			}

			for _, matches := range findResult {
				switch string(matches[1]) {
				case "$":
					name := string(matches[2])
					v, exists := env[name]
					if !exists {
						withWarn = true
						log.Message(log.WARNING, `Undefined environment variable "%s" in line %d, using empty value`, name, *lineNumber)
						v = []byte("")
					}
					line = bytes.Replace(line, matches[0], v, -1)

				case "#":
					s := string(matches[2])
					if strings.HasPrefix(s, "include ") {
						b := new(bytes.Buffer)
						p := strings.Split(s, " ")
						if len(p) != 2 {
							msgs.Add(fmt.Sprintf(`Illegal preprocessor command "%s" in line %d`, string(matches[2]), *lineNumber))
						} else {
							var err error
							repl, fn, err := readFile(p[1], base)
							if err != nil {
								msgs.Add(fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), *lineNumber))
							} else {
								*lineNumber--
								w := false
								b, w, err = populate(repl, filepath.Dir(fn), lineNumber)
								if w {
									withWarn = true
								}
								if err != nil {
									msgs.Add(fmt.Sprintf(`Include error "%s" in line %d`, err.Error(), *lineNumber))
								}
							}
						}
						line = bytes.Replace(line, matches[0], bytes.TrimSpace(b.Bytes()), -1)
						continue
					}

					msgs.Add(fmt.Sprintf(`Unknown preprocessor command "%s" in line %d`, string(matches[2]), *lineNumber))
				}
			}
		}

		if len(line) != 0 {
			newData.Write(bytes.TrimSpace(line))
			newData.WriteByte(byte('\n'))

		}
	}

	err = msgs.Error()

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// LoadFile parses the specified file into a Config object
func LoadFile(fileName string, cfg any) (err error) {
	if len(env) == 0 {
		loadEnv()
	}

	data, fn, err := readFile(fileName, misc.AppWorkDir())
	if err != nil {
		return
	}

	withWarn := false

	var newData *bytes.Buffer

	defer func() {
		if err != nil || withWarn {
			msg := new(bytes.Buffer)
			msg.WriteString("Config file:\n>>>\n")
			lines := bytes.Split(newData.Bytes(), []byte("\n"))
			for i, line := range lines {
				msg.WriteString(fmt.Sprintf("%04d | %s\n", i+1, bytes.TrimSpace(line)))
			}
			msg.WriteString("<<<")

			log.Message(log.WARNING, "%s", msg.String())
		}
	}()

	newData, withWarn, err = populate(data, filepath.Dir(fn), nil)
	if err != nil {
		return
	}

	data = newData.Bytes()
	configText = string(data)

	err = toml.Unmarshal(data, cfg)
	if err != nil {
		return
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
					SetCommon(f)
					break
				}
			}
		}
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// GetConfig --
func GetConfig() any {
	return fullConfig
}

//----------------------------------------------------------------------------------------------------------------------------//

// SetCommon --
func SetCommon(cc *Common) {
	commonConfig = cc
}

// GetCommon --
func GetCommon() *Common {
	return commonConfig
}

//----------------------------------------------------------------------------------------------------------------------------//

var (
	stdReplaces = map[string]string{
		`(password\s*=\s*")(.*)(")`: `$1*$3`,
		`(secret\s*=\s*")(.*)(")`:   `$1*$3`,
		`(users\s*=\s*{)(.*)(})`:    `$1*$3`,
	}

	replace = misc.NewReplace()
)

func init() {
	err := replace.AddMulti(stdReplaces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config.init: %s", err.Error())
		os.Exit(misc.ExProgrammerError)
	}
}

// AddFilter --
func AddFilter(re string, replaceTo string) error {
	return replace.Add(re, replaceTo)
}

// GetText -- get prepared configuration text
func GetText() string {
	return configText
}

// GetSecuredText -- get prepared configuration text with securing
func GetSecuredText() string {
	return replace.Do(configText)
}

//----------------------------------------------------------------------------------------------------------------------------//

func loadEnv() {
	env = map[string][]byte{
		"___AppPID":      []byte(strconv.FormatInt(int64(syscall.Getpid()), 10)),
		"___AppVersion":  []byte(misc.AppVersion()),
		"___AppTags":     []byte(misc.AppTags()),
		"___Copyright":   []byte(misc.Copyright()),
		"___BuildTime":   []byte(misc.BuildTime()),
		"___AppName":     []byte(misc.AppName()),
		"___AppFullName": []byte(misc.AppFullName()),
		"___AppExecPath": []byte(misc.AppExecPath()),
		"___AppExecName": []byte(misc.AppExecName()),
		"___AppWorkDir":  []byte(misc.AppWorkDir()),
	}

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
