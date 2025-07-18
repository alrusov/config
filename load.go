package config

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"maps"
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
	configText     = ""
	fullConfig     = any(nil)
	commonConfig   *Common
	listenerConfig *Listener

	rePreprocessor = regexp.MustCompile(`(\$\{|\{\$|\{#|\{@)([^\}]+)(?:\})`)

	// Use the # symbol at the begining of the line for comment
	reComment = regexp.MustCompile(`(?m)^\s*#.*$`)

	// Use the \ symbol at the end line to continue to next line
	reMultiLine = regexp.MustCompile(`(?m)\s*\\\s*\r?\n\s*`)

	fEnv   = os.Environ
	env    = map[string][]byte{}
	sysEnv = map[string][]byte{
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
)

//----------------------------------------------------------------------------------------------------------------------------//

var embedFS *embed.FS

func Embed(fs *embed.FS) {
	embedFS = fs
}

//----------------------------------------------------------------------------------------------------------------------------//

func readFile(name string, base string, mandatory bool) ([]byte, string, error) {
	var err error
	f := fs.File(nil)

	if embedFS != nil {
		f, _ = embedFS.Open(name)
	}

	if f == nil {
		// embedFS is nil or the embedded file was not found - let's try reading from the file system

		name, err = misc.AbsPathEx(name, base)
		if err != nil {
			return nil, "", err
		}

		f, err = os.Open(name)

		if err != nil {
			if mandatory {
				return nil, name, err
			}

			log.Message(log.NOTICE, "Included file %s not found", name)
			return nil, name, nil
		}
	}

	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, name, nil
	}

	data = bytes.TrimSpace(reComment.ReplaceAll(data, []byte{}))
	data = bytes.TrimSpace(reMultiLine.ReplaceAll(data, []byte{' '}))
	data = bytes.TrimRight(data, "\\")

	return data, name, nil
}

// ----------------------------------------------------------------------------------------------------------------------------//

type populate struct {
	lineNumber uint
	macroses   map[string][]byte
}

func (populate *populate) do(data []byte, base string) (newData *bytes.Buffer, withWarn bool, err error) {
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

		populate.lineNumber++

		line = bytes.ReplaceAll(line, []byte("\t"), []byte(" "))

		if line[0] == '@' {
			m := bytes.SplitN(line, []byte("="), 2)
			for i, s := range m {
				m[i] = bytes.TrimSpace(s)
			}

			if len(m) < 2 || len(m[0]) == 1 {
				msgs.Add(`Bad macros "%s" in line %d`, line, populate.lineNumber)
				continue
			}

			populate.macroses[string(m[0][1:])] = m[1]
			continue
		}

		nIter := 0

		for {
			nIter++
			if nIter > 32 {
				msgs.Add(`Too many iterations for "%s" in line %d`, line, populate.lineNumber)
				break
			}

			findResult := rePreprocessor.FindAllSubmatch(line, -1)
			if len(findResult) == 0 {
				break
			}

			for _, matches := range findResult {
				switch string(matches[1]) {
				case "${", "{$":
					name := string(matches[2])
					v, exists := env[name]
					if !exists {
						withWarn = true
						log.Message(log.WARNING, `Undefined environment variable "%s" in line %d, using empty value`, name, populate.lineNumber)
						v = []byte("")
					}
					line = bytes.Replace(line, matches[0], v, -1)

				case "{@":
					name := string(matches[2])
					v, exists := populate.macroses[name]
					if !exists {
						msgs.Add(`Undefined macros "%s" in line %d`, name, populate.lineNumber)
						v = []byte("")
					}
					line = bytes.Replace(line, matches[0], v, -1)

				case "{#":
					s := string(matches[2])
					if strings.HasPrefix(s, "include ") || strings.HasPrefix(s, "#include ") {
						mandatory := s[0] != '#'
						b := new(bytes.Buffer)
						p := strings.SplitN(s, " ", 2)
						if len(p) != 2 {
							msgs.Add(`Illegal preprocessor command "%s" in line %d`, string(matches[2]), populate.lineNumber)
						} else {
							var err error
							repl, fn, err := readFile(p[1], base, mandatory)
							if err != nil {
								msgs.Add(`Include error "%s" in line %d`, err.Error(), populate.lineNumber)
							} else {
								populate.lineNumber--
								w := false
								b, w, err = populate.do(repl, filepath.Dir(fn))
								if w {
									withWarn = true
								}
								if err != nil {
									msgs.Add(`Include error "%s" in line %d`, err.Error(), populate.lineNumber)
								}
							}
						}
						line = bytes.Replace(line, matches[0], bytes.TrimSpace(b.Bytes()), -1)
						continue
					}

					msgs.Add(`Unknown preprocessor command "%s" in line %d`, string(matches[2]), populate.lineNumber)
					line = []byte{}
				}
			}
		}

		newData.Write(bytes.TrimSpace(line))
		newData.WriteByte(byte('\n'))
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

	data, fn, err := readFile(fileName, misc.AppWorkDir(), true)
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

	populate := &populate{
		macroses:   make(map[string][]byte, 128),
		lineNumber: 0,
	}

	newData, withWarn, err = populate.do(data, filepath.Dir(fn))
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

	lookingForStdBlocks(cfg)

	return
}

func lookingForStdBlocks(cfg any) {
	c := reflect.ValueOf(cfg)

	if c.Kind() == reflect.Ptr {
		c = c.Elem()
	}

	if c.Kind() == reflect.Struct {
		ft := c.Type()
		fCnt := c.NumField()

		for i := range fCnt {
			f := c.Field(i)

			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}

			if f.Kind() != reflect.Struct {
				continue
			}

			t := ft.Field(i)
			if !t.IsExported() {
				continue
			}

			switch f := f.Interface().(type) {
			default:
				lookingForStdBlocks(f)
			case Common:
				SetCommon(&f)
			case Listener:
				SetListener(&f)
			}
		}
	}
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

// SetListener --
func SetListener(cc *Listener) {
	listenerConfig = cc
}

// GetCommon --
func GetListener() *Listener {
	return listenerConfig
}

// ----------------------------------------------------------------------------------------------------------------------------//
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
	osEnv := fEnv()
	env = make(map[string][]byte, len(osEnv)+len(sysEnv))
	maps.Copy(env, sysEnv)

	for _, s := range osEnv {
		df := strings.SplitN(s, "=", 2)
		df[0] = strings.TrimSpace(df[0])

		if _, exists := sysEnv[df[0]]; exists {
			continue
		}

		v := ""
		if len(df) > 1 {
			v = strings.TrimSpace(df[1])
		}
		env[df[0]] = []byte(v)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
