package config

import (
	"strings"
	"testing"
)

//----------------------------------------------------------------------------------------------------------------------------//

// SecureText -- те же правила, что у дампа конфига, для журналов: тела JSON и URI с query
func TestSecureTextJSON(t *testing.T) {
	cases := []struct {
		name string
		src  string
		dst  string
	}{
		{"flat", `{"name":"line","password":"p1-secret","host":"h"}`, `{"name":"line","password":"*","host":"h"}`},
		{"spaces", `{ "password" : "p1-secret" }`, `{ "password" : "*" }`},
		{"case and suffix", `{"consumerPassword":"p1","networkPassword":"p2","clientSecret":"s3","consumerToken":"t4","api-key":"k5"}`,
			`{"consumerPassword":"*","networkPassword":"*","clientSecret":"*","consumerToken":"*","api-key":"*"}`},
		{"escaped quote", `{"password":"a\"b\\","x":"y"}`, `{"password":"*","x":"y"}`},
		{"nested", `[{"parameters":{"password":"enc:v2:abc","consumerUseToken":true,"timeout":5}}]`,
			`[{"parameters":{"password":"*","consumerUseToken":true,"timeout":5}}]`},
		{"empty stays shape", `{"password":""}`, `{"password":"*"}`},
		{"not a suffix", `{"passwordHint":"visible","tokenizer":"visible"}`, `{"passwordHint":"visible","tokenizer":"visible"}`},
		{"non-string value", `{"password":null,"secret":12345}`, `{"password":null,"secret":12345}`},
	}

	for _, c := range cases {
		if got := SecureText(c.src); got != c.dst {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.dst)
		}
	}
}

func TestSecureTextQuery(t *testing.T) {
	cases := []struct {
		name string
		src  string
		dst  string
	}{
		{"token", `GET "/api/x?id=1&token=t2-secret&y=2" from 10.0.0.1`, `GET "/api/x?id=1&token=*&y=2" from 10.0.0.1`},
		{"first param", `/api/x?access-token=t2-secret#frag`, `/api/x?access-token=*#frag`},
		// password=… дополнительно ловит правило DSN key=value (до пробела или кавычки): хвост query
		// после пароля скрывается вместе с ним -- осознанная избыточность
		{"password", `/api/x?user=u&password=p1-secret&y=2 x`, `/api/x?user=u&password=* x`},
		{"plain", `/api/x?id=1&name=a`, `/api/x?id=1&name=a`},
	}

	for _, c := range cases {
		if got := SecureText(c.src); got != c.dst {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.dst)
		}
	}
}

// Ключ, которого нет в списке по умолчанию, добавляется для TOML, JSON и query разом
func TestAddSecretKey(t *testing.T) {
	src := "communityName = \"c-toml\"\n{\"communityName\":\"c-json\"}\n/api?communityName=c-query&x=1"

	if got := SecureText(src); !strings.Contains(got, "c-json") {
		t.Fatalf("communityName must not be masked before AddSecretKey: %s", got)
	}

	saved := *replace
	t.Cleanup(func() { *replace = saved })

	if err := AddSecretKey(`communityName`); err != nil {
		t.Fatal(err)
	}

	got := SecureText(src)
	for _, v := range []string{"c-toml", "c-json", "c-query"} {
		if strings.Contains(got, v) {
			t.Errorf("%q is not masked: %s", v, got)
		}
	}
	if !strings.Contains(got, "&x=1") {
		t.Errorf("query tail lost: %s", got)
	}
}

// Дамп конфига и журналы -- один маскировщик
func TestSecureTextSameAsSecuredText(t *testing.T) {
	saved := configText
	t.Cleanup(func() { configText = saved })

	configText = "dsn = \"postgres://core:p1-secret@db:5432/rtdb\"\njwt-secret = \"s2-secret\"\n"
	if GetSecuredText() != SecureText(configText) {
		t.Errorf("GetSecuredText and SecureText differ")
	}
	for _, v := range []string{"p1-secret", "s2-secret"} {
		if strings.Contains(SecureText(configText), v) {
			t.Errorf("%q is not masked: %s", v, SecureText(configText))
		}
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
