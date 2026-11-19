package config

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/alrusov/log"
)

//----------------------------------------------------------------------------------------------------------------------------//

// isolateEnv -- LoadFile кэширует окружение в env при первом вызове, а TestPopulate подменяет источник fEnv.
// До теста -- настоящее окружение и пустой кэш (виден t.Setenv), после -- прежний fEnv и пустой кэш,
// чтобы следующий тест загрузил своё окружение в любом порядке запуска (-shuffle)
func isolateEnv(t *testing.T) {
	prevFEnv := fEnv
	fEnv = os.Environ
	env = map[string][]byte{}

	t.Cleanup(func() {
		fEnv = prevFEnv
		env = map[string][]byte{}
	})
}

//----------------------------------------------------------------------------------------------------------------------------//

// IDC: маскировщик закрывает пароль в DSN (URL и key=value), все виды кавычек TOML, *token/*key/*salt --
// и не трогает несекретные части строки. Секреты синтетические, у каждого случая свой
func TestSecuredTextMasksDsnAndTokens(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		secret string
		keep   []string
	}{
		{"url-dsn", `db-dsn = "postgres://mdl:SynUrlPw1@db.local:5432/mdl?sslmode=disable"`, "SynUrlPw1", []string{"postgres://mdl:", "@db.local:5432/mdl?sslmode=disable"}},
		{"url-in-any-key", `webapi-url = "https://svc:SynAnyKeyPw1@pi.local/piwebapi"`, "SynAnyKeyPw1", []string{"https://svc:", "@pi.local/piwebapi"}},
		{"url-empty-user", `cache = "redis://:SynRedisPw1@redis:6379/0"`, "SynRedisPw1", []string{"redis://:", "@redis:6379/0"}},
		{"url-single-quote", `db-dsn = "postgres://mdl:Syn'QuotePw1@db:5432/mdl"`, "Syn'QuotePw1", []string{"postgres://mdl:", "@db:5432/mdl"}},
		{"url-escaped-dquote", `db-dsn = "postgres://mdl:Syn\"DquotePw1@db:5432/mdl"`, `Syn\"DquotePw1`, []string{"postgres://mdl:", "@db:5432/mdl"}},
		{"url-space", `db-dsn = "postgres://mdl:Syn SpacePw1@db:5432/mdl"`, "Syn SpacePw1", []string{"postgres://mdl:", "@db:5432/mdl"}},
		{"url-raw-at", `db-dsn = "postgres://mdl:Syn@AtPw1@db:5432/mdl"`, "Syn@AtPw1", []string{"postgres://mdl:", "@db:5432/mdl"}},
		{"kv-dsn", `db-dsn = "host=db.local user=mdl password=SynKvPw1 dbname=mdl"`, "SynKvPw1", []string{"host=db.local user=mdl password=", " dbname=mdl"}},
		{"kv-semicolon", `db-dsn = "host=db password=Syn;SemiPw1 dbname=mdl"`, "Syn;SemiPw1", []string{"host=db password=", " dbname=mdl"}},
		{"kv-ado", `mssql = "Server=db;User Id=sa;Password=SynAdoPw1;Database=x"`, "SynAdoPw1", []string{"Server=db;User Id=sa;Password="}},
		{"kv-ado-escaped", `mssql = "Server=db;User Id=sa;Password=\"Syn;AdoEscPw1\";Database=x"`, `Syn;AdoEscPw1`, []string{"Server=db;User Id=sa;Password="}},
		{"kv-pwd", `mssql = "uid=sa;pwd=SynPwdPw1;db=x"`, "SynPwdPw1", []string{"uid=sa;pwd="}},
		{"kv-suffix", `x = "db_password=SynSuffixPw1"`, "SynSuffixPw1", []string{"db_password="}},
		{"dq-std", `password = "SynStdPw1"`, "SynStdPw1", []string{`password = "`}},
		{"dq-escaped", `password = "Syn\"EscPw1"`, `Syn\"EscPw1`, []string{`password = "`}},
		{"dq-triple", `password = """SynTripleDqPw1"""`, "SynTripleDqPw1", []string{`password = """`}},
		{"sq", `password = 'SynSqPw1'`, "SynSqPw1", []string{"password = '"}},
		{"sq-triple", `password = '''SynTripleSqPw1'''`, "SynTripleSqPw1", []string{"password = '''"}},
		{"token", `access-token = "SynTokenPw1"`, "SynTokenPw1", []string{`access-token = "`}},
		{"api-key", `api-key = "SynApiKeyPw1"`, "SynApiKeyPw1", []string{`api-key = "`}},
		{"quoted-key", `"api-key" = "SynQuotedKeyPw1"`, "SynQuotedKeyPw1", []string{`"api-key" = "`}},
		{"salt", `salt = "SynSaltPw1"`, "SynSaltPw1", []string{`salt = "`}},
		{"secret-inline", `jwt = { secret = "SynJwtPw1", lifetime = 10 }`, "SynJwtPw1", []string{`jwt = { secret = "`, `", lifetime = 10 }`}},
		{"users", `users = {"u1" = "SynUsersPw1", "u2" = "SynUsersPw2"}`, "SynUsersPw1 SynUsersPw2", []string{"users = {"}},
		{"users-brace", `users = {"u1" = "Syn}BracePw1"}`, "Syn}BracePw1", []string{"users = {"}},
		{"url-token-no-colon", `remote = "https://SynGlTokenPw1@gitlab.local/idc/x.git"`, "SynGlTokenPw1", []string{`remote = "https://`, "@gitlab.local/idc/x.git"}},
		{"url-token-no-colon-port", `remote = "https://SynGlTokenPw2@gitlab.local:8443/x"`, "SynGlTokenPw2", []string{"https://", "@gitlab.local:8443/x"}},
		// многострочные тройные кавычки: сырой текст и WA-дамп LoadFile ("%04d | " в начале каждой строки);
		// внутри значения -- " ' и \" (ревью main 26.09 00:37)
		{"dq-triple-multiline", "password = \"\"\"SynMlDqA1\n\"SynMlDqB2\" and 'SynMlDqC3' \\\"q\\\"\nSynMlDqD4\"\"\"\nname = \"x\"",
			"SynMlDqA1 SynMlDqB2 SynMlDqC3 SynMlDqD4", []string{`password = """`, `name = "x"`}},
		{"sq-triple-multiline", "password = '''SynMlSqA1\n\"SynMlSqB2\"\nSynMlSqC3'''\nname = 'x'",
			"SynMlSqA1 SynMlSqB2 SynMlSqC3", []string{"password = '''", "name = 'x'"}},
		{"dq-triple-multiline-wa", "0001 | password = \"\"\"SynMlWaA1\n0002 | 'SynMlWaB2' \\\"SynMlWaC3\\\"\n0003 | SynMlWaD4\"\"\"\n0004 | name = \"x\"\n",
			"SynMlWaA1 SynMlWaB2 SynMlWaC3 SynMlWaD4", []string{`0001 | password = """`, `0004 | name = "x"`}},
		{"sq-triple-multiline-wa", "0001 | password = '''SynMlWaSqA1\n0002 | SynMlWaSqB2'''\n0003 | name = 'x'\n",
			"SynMlWaSqA1 SynMlWaSqB2", []string{"0001 | password = '''", "0003 | name = 'x'"}},
		// незакрытая тройная в сообщении populate по одной строке -- маска до конца, а не пустое ""
		{"dq-triple-unclosed-line", `Bad macros "password = """SynOpenPw1" in line 5`, "SynOpenPw1", []string{`Bad macros "password = """`}},
	}

	for _, c := range cases {
		got := replace.Do(c.src)

		if strings.Contains(got, c.secret) {
			t.Errorf("%s: secret is not masked", c.name)
		}

		// ни один кусок секрета не должен пережить маску: ленивый квантификатор под (?U)
		// снял бы один символ, граница по ' " ; @ пробелу -- только часть значения
		for _, piece := range regexp.MustCompile(`[^A-Za-z0-9]+`).Split(c.secret, -1) {
			if len(piece) >= 3 && strings.Contains(got, piece) {
				t.Errorf("%s: a piece of the secret (%d chars) survived the mask", c.name, len(piece))
			}
		}

		for _, k := range c.keep {
			if !strings.Contains(got, k) {
				t.Errorf("%s: non-secret part %q is lost: %q", c.name, k, got)
			}
		}
	}

	// несекретное маскировщик не трогает
	for _, src := range []string{
		`listener = { bind-addr = ":1234", root = "/opt/x", proxy-prefix = "/config-test/" }`,
		`webapi-url = "https://pi.local:8443/piwebapi"`,
		`db-dsn = "host=db.local user=mdl dbname=mdl sslmode=disable"`,
		`name = "password" # слово в значении, не ключ`,
		`remote = "https://gitlab.local/idc/x.git"`,
		`module = "https://github.com/alrusov/misc@v1.1.36"`,
	} {
		if got := replace.Do(src); got != src {
			t.Errorf("non-secret line is changed: %q -> %q", src, got)
		}
	}
}

// IDC: DSN из переменной окружения не выходит в WA-дамп конфига при предупреждении загрузки
func TestWarningDumpMasksDsnFromEnv(t *testing.T) {
	const pw = "IdcEnvDsnPassw0rd"

	t.Setenv("IDC_CONFIG_TEST_DSN", "postgres://mdl:"+pw+"@db.local:5432/mdl")

	isolateEnv(t)

	dir := t.TempDir()
	fn := filepath.Join(dir, "dsn.toml")
	src := `x = "${IDC_CONFIG_TEST_UNDEFINED_VAR}"
db-dsn = "${IDC_CONFIG_TEST_DSN}"
`
	if err := os.WriteFile(fn, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	log.SetConsoleWriter(buf)
	defer log.SetConsoleWriter(os.Stderr)

	cfg := struct {
		X   string `toml:"x"`
		DSN string `toml:"db-dsn"`
	}{}

	if err := LoadFile(fn, &cfg); err != nil {
		t.Fatal(err)
	}

	out := buf.String()

	if !strings.Contains(out, "Config file:") {
		t.Fatalf("no config dump in the log, the test does not reach the dump path")
	}

	if !strings.Contains(out, "@db.local:5432/mdl") {
		t.Fatalf("env is not substituted in the dump, the test does not check the DSN path")
	}

	if strings.Contains(out, pw) {
		t.Errorf("config dump contains the DSN password (%d chars)", len(pw))
	}

	if !strings.Contains(cfg.DSN, pw) {
		t.Errorf("masking must not touch loaded values")
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// IDC: ошибка «Too many iterations» несёт строку с уже подставленными переменными окружения -- через маскировщик
func TestTooManyIterationsMessageIsSecured(t *testing.T) {
	const pw = "IdcLoopPassw0rd"

	t.Setenv("IDC_CONFIG_TEST_LOOP", "postgres://mdl:"+pw+"@db.local/mdl ${IDC_CONFIG_TEST_LOOP}")
	isolateEnv(t)

	dir := t.TempDir()
	fn := filepath.Join(dir, "loop.toml")
	if err := os.WriteFile(fn, []byte(`db-dsn = "${IDC_CONFIG_TEST_LOOP}"`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	log.SetConsoleWriter(buf)
	defer log.SetConsoleWriter(os.Stderr)

	cfg := struct {
		DSN string `toml:"db-dsn"`
	}{}

	err := LoadFile(fn, &cfg)
	if err == nil || !strings.Contains(err.Error(), "Too many iterations") {
		t.Fatalf("no iteration error, the test does not reach the message path")
	}

	if !strings.Contains(err.Error(), "@db.local/mdl") {
		t.Fatalf("env is not substituted in the message, the test does not check the masking")
	}

	if strings.Contains(err.Error(), pw) {
		t.Errorf("iteration error contains the DSN password (%d chars)", len(pw))
	}

	if strings.Contains(buf.String(), pw) {
		t.Errorf("config dump contains the DSN password (%d chars)", len(pw))
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// IDC: сообщение «Bad macros» несёт строку конфига как есть -- через маскировщик
func TestBadMacrosMessageIsSecured(t *testing.T) {
	const pw = "SynMacroPw1"

	isolateEnv(t)

	fn := filepath.Join(t.TempDir(), "macro.toml")
	if err := os.WriteFile(fn, []byte("@db-dsn postgres://mdl:"+pw+"@db.local/mdl\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	log.SetConsoleWriter(buf)
	defer log.SetConsoleWriter(os.Stderr)

	cfg := struct{}{}

	err := LoadFile(fn, &cfg)
	if err == nil || !strings.Contains(err.Error(), "Bad macros") {
		t.Fatalf("no bad macros error, the test does not reach the message path")
	}

	if !strings.Contains(err.Error(), "@db.local/mdl") {
		t.Fatalf("the line is not in the message, the test does not check the masking")
	}

	if strings.Contains(err.Error(), pw) || strings.Contains(buf.String(), pw) {
		t.Errorf("bad macros error or dump contains the password (%d chars)", len(pw))
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

// IDC: известные ограничения маски -- зафиксированы тестом, чтобы смена поведения была видна
// (ревью main 26.09 00:37, NIT). Пароль с сырым / в URL остаётся открытым: граница по /, как у net/url
func TestKnownMaskLimitations(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`cache = "redis://u:Sl/ash@h:6379"`, `cache = "redis://u:Sl/ash@h:6379"`},
		{`url = "http://h:8080" # me@x.org`, `url = "http://h:*@x.org`},
		{`url = "https://user@host/x"`, `url = "https://*@host/x"`},
	} {
		if got := replace.Do(c.src); got != c.want {
			t.Errorf("known behaviour changed: %q -> %q, want %q", c.src, got, c.want)
		}
	}
}

// IDC: многострочная тройная кавычка и токен в URL без двоеточия не выходят в настоящий WA-дамп
// LoadFile (строки с префиксом "%04d | ", маска по дампу целиком) -- ревью main 26.09 00:37
func TestWarningDumpMasksMultilineAndUrlToken(t *testing.T) {
	isolateEnv(t)

	dir := t.TempDir()
	fn := filepath.Join(dir, "ml.toml")
	src := "x = \"${IDC_CONFIG_TEST_UNDEFINED_VAR}\"\n" +
		"password = \"\"\"IdcMlFirstPw1\nIdcMlSecondPw2 IdcMlThirdPw3\"\"\"\n" +
		"remote = \"https://IdcUrlTokenPw4@gitlab.local/idc/x.git\"\n"
	if err := os.WriteFile(fn, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	log.SetConsoleWriter(buf)
	defer log.SetConsoleWriter(os.Stderr)

	cfg := struct {
		X        string `toml:"x"`
		Password string `toml:"password"`
		Remote   string `toml:"remote"`
	}{}

	if err := LoadFile(fn, &cfg); err != nil {
		t.Fatal(err)
	}

	out := buf.String()

	if !strings.Contains(out, "Config file:") || !strings.Contains(out, "@gitlab.local/idc/x.git") {
		t.Fatalf("no config dump in the log, the test does not reach the dump path")
	}

	for _, pw := range []string{"IdcMlFirstPw1", "IdcMlSecondPw2", "IdcMlThirdPw3", "IdcUrlTokenPw4"} {
		if strings.Contains(out, pw) {
			t.Errorf("config dump contains a secret (%d chars)", len(pw))
		}
		if !strings.Contains(cfg.Password+cfg.Remote, pw) {
			t.Errorf("masking must not touch loaded values")
		}
	}
}
