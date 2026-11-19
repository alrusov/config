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
	defer msgs.Free()

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
				msgs.Add(`Bad macros "%s" in line %d`, replace.Do(string(line)), populate.lineNumber)
				continue
			}

			populate.macroses[string(m[0][1:])] = m[1]
			continue
		}

		nIter := 0

		for {
			nIter++
			if nIter > 32 {
				// IDC: строка уже с частично подставленными переменными окружения -- через маскировщик
				msgs.Add(`Too many iterations for "%s" in line %d`, replace.Do(string(line)), populate.lineNumber)
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

			// IDC: текст уже после подстановки переменных окружения -- тот же маскировщик, что у GetSecuredText,
			// иначе пароли и секреты уходят в лог открытым текстом при любом предупреждении загрузки
			log.Message(log.WARNING, "%s", replace.Do(msg.String()))
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
	// IDC: маскировщик дампа конфига (GetSecuredText, WA-дамп LoadFile, сообщения populate).
	//
	// Все шаблоны компилируются с (?iU) (misc.Replace.Add): под U квантификатор ЛЕНИВЫЙ, поэтому
	// значения берутся жадно явно -- (?-U:...), иначе маскируется один символ или до первой кавычки.
	// Порядок детерминирован (срез, не map): правила применяются друг за другом по всему тексту.
	//
	// Ограничения (осознанно): ключи вне списка (pass, auth, bearer, *-credentials); значения
	// в массивах (tokens = ["…"]) и числа без кавычек (secret = 98765432); сырой / в пароле URL
	// (redis://u:Sl/ash@h -- граница пароля по /, как у net/url, пароль остаётся открытым);
	// избыточная маска несекретных *key (sort-key) и имени пользователя URL без пароля
	// (https://user@host), а также хвоста до @ в комментарии (url = "http://h:8080" # me@x.org) --
	// только в дампе, загруженные значения не трогаются.
	stdReplaces = append(keyReplaces(secretKey), [][2]string{
		// однострочная inline-таблица пользователей -- до ПОСЛЕДНЕЙ } в строке
		{`(users\s*=\s*{)((?-U:.*))(})`, `$1*$3`},

		// пароль в DSN вида key=value без кавычек (libpq "password=...", ADO "Password=...;",
		// "pwd=..."): ; в libpq -- обычный символ, \" -- кавычка ADO внутри TOML-строки;
		// цена -- маска хвоста ADO-строки до пробела/кавычки
		{`((?:password|passwd|pwd)\s*=\s*)((?-U:(?:[^\s"'\\]|\\.)+))`, `${1}*`},

		// пароль в userinfo URL -- любой ключ (db-dsn, *-url): user без : и @ (может быть пустым),
		// пароль -- жадно до ПОСЛЕДНЕГО @ перед путём (как net/url), включая ' " пробел @
		{`(://(?-U:[^/@\s:]*):)((?-U:[^/\n]*))(@)`, `$1*$3`},

		// userinfo БЕЗ двоеточия (https://<токен>@host -- токен в remote-URL): всё до последнего @
		// перед хостом; user:pass сюда не попадает -- после правила выше там уже ":*@"
		{`(://)((?-U:[^/\s:]+))(@)`, `$1*$3`},
	}...)

	replace = misc.NewReplace()
)

// secretKey -- имя ключа с секретом: *password, *passwd, *pwd, *secret, *token, *key, *salt
// (в TOML -- и ключ в кавычках: "api-key" = ..., закрывающая кавычка -- в keyReplaces)
const secretKey = `[\w.-]*(?:password|passwd|pwd|secret|token|key|salt)`

// keyReplaces -- правила для значения ключа с секретом (key -- регулярное выражение имени ключа):
//
//   - TOML: значение во всех кавычках TOML -- ОДНИМ правилом: альтернатива в Go выбирается слева направо
//     в одной позиции (тройные раньше одинарных), и уже замаскированное не проходится повторно; '…' по TOML
//     без escape, в "…" \" -- часть значения. Тройные -- МНОГОСТРОЧНЫЕ: (?s) пропускает перевод строки
//     (WA-дамп маскируется целиком, вместе с префиксами "%04d | " внутри значения); незакрытая тройная
//     (сообщение populate по одной строке) -- маска до конца текста, иначе "…" взяла бы пустое "" и оставила хвост;
//   - JSON: "key": "…" (тела HTTP-запросов в журнале); \" внутри значения -- часть значения;
//   - query: ?key=… и &key=… (URI в журнале) -- до & пробела кавычки #.
func keyReplaces(key string) [][2]string {
	return [][2]string{
		{`(` + key + `["']?\s*=\s*)(?-U:(""")(?s:.*?)(?:"""|\z)|(''')(?s:.*?)(?:'''|\z)|(")(?:[^"\\\n]|\\.)*"|(')[^'\n]*')`,
			`${1}${2}${3}${4}${5}*${2}${3}${4}${5}`},
		{`("` + key + `"\s*:\s*)(?-U:"(?:[^"\\]|\\.)*")`, `${1}"*"`},
		{`([?&]` + key + `=)((?-U:[^&\s"'#]+))`, `${1}*`},
	}
}

// AddSecretKey -- дополнительное имя ключа с секретом (регулярное выражение, например `communityName`):
// его значения маскируются в TOML, JSON и query так же, как у ключей по умолчанию
func AddSecretKey(key string) error {
	for _, r := range keyReplaces(key) {
		if err := replace.Add(r[0], r[1]); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	for _, r := range stdReplaces {
		if err := replace.Add(r[0], r[1]); err != nil {
			fmt.Fprintf(os.Stderr, "config.init: %s", err.Error())
			os.Exit(misc.ExProgrammerError)
		}
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

// SecureText -- замаскировать секреты в произвольном тексте теми же правилами, что GetSecuredText
// (журналы: URI, заголовки, тела HTTP-запросов)
func SecureText(s string) string {
	return replace.Do(s)
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

func Env() map[string][]byte {
	// unsafe
	return env
}

//----------------------------------------------------------------------------------------------------------------------------//
