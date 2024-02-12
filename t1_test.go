package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alrusov/jsonw"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

type testS1 struct {
	X int    `toml:"x"`
	Y string `toml:"y"`
}

type testHTTP struct {
	Listener Listener `toml:"listener"`
}

type testBasicOptions struct {
}

type testJwtOptions struct {
	Secret   string `toml:"secret"`
	Lifetime int    `toml:"lifetime"`
}

type testCfg struct {
	P0           string            `toml:"param-0"`
	P1           string            `toml:"param-1"`
	P2           int               `toml:"param-2"`
	P3           string            `toml:"param-3"`
	P4           int               `toml:"param-4"`
	P5           misc.InterfaceMap `toml:"param-5"`
	Duration1    Duration          `toml:"duration1"`
	Duration2    Duration          `toml:"duration2"`
	Duration3    Duration          `toml:"duration3"`
	Duration4    Duration          `toml:"duration4"`
	Plast        testS1            `toml:"param-last"`
	FromMacroses string            `toml:"from-macroses"`
	HTTP         testHTTP          `toml:"http"`
}

func (options *testBasicOptions) Check(cfg any) (err error) {
	return
}

func (options *testJwtOptions) Check(cfg any) (err error) {
	return
}

func TestPopulate(t *testing.T) {
	fEnv = func() []string {
		return []string{
			"ENV1=VAL1",
			"ENV2=666",
		}
	}

	iconFile, _ := misc.AbsPath("/tmp/favicon.ico") // workaround for idiotic windows

	expected := testCfg{
		P0: "***",
		P1: "VAL1",
		P2: 666,
		P3: `!@#$%^&@ qwertyuiop asdfghjkl 123456789 ZZZ`,
		P4: 123456,
		P5: misc.InterfaceMap{
			"field1": "val1",
			"field2": 777,
		},
		Duration1: 100 * Duration(time.Nanosecond),
		Duration2: 5 * Duration(time.Hour*24),
		Duration3: 10 * Duration(time.Microsecond),
		Duration4: 10 * Duration(time.Microsecond),
		Plast: testS1{
			X: 1,
			Y: "Y",
		},

		FromMacroses: "- 123456 - qwerty asdfgh -",

		HTTP: testHTTP{
			Listener: Listener{
				Addr:                   ":1234",
				SSLCombinedPem:         "",
				Timeout:                Duration(6 * time.Second),
				Root:                   "",
				ProxyPrefix:            "/config-test",
				IconFile:               iconFile,
				DisabledEndpointsSlice: []string{"/aaa*", "!/aaa/bbb"},
				DisabledEndpoints:      misc.BoolMap{"/aaa*": true, "!/aaa/bbb": true},
				Auth: Auth{
					EndpointsSlice: map[string][]string{
						"/xxx/": {" *       "},
						"/yyy":  {" user1 ", " user2", " @group1  ", " ! @group2  ", " !user3", "!", " ! "},
					},
					Endpoints: map[string]misc.BoolMap{
						"/xxx": {"*": true},
						"/yyy": {"user1": true, "user2": true, "@group1": true, "@group2": false, "user3": false},
					},
					UsersMap: misc.StringMap{"test-user1": "pwd1", "test-user2": "pwd2", "test-user3@   g0  ": "pwd3", "test-user4@g1": "pwd4", "test-user5  @  g1,g2,g3, g5@xxx , @g6@  ": "pwd5"},
					Users: map[string]User{
						"test-user1": {Password: "pwd1", Groups: []string{}},
						"test-user2": {Password: "pwd2", Groups: []string{}},
						"test-user3": {Password: "pwd3", Groups: []string{"g0"}},
						"test-user4": {Password: "pwd4", Groups: []string{"g1"}},
						"test-user5": {Password: "pwd5", Groups: []string{"g1", "g2", "g3", "g5@xxx", "@g6@"}},
					},
					Methods: map[string]*AuthMethod{
						"basic": {
							Enabled: true,
							Score:   0,
							Options: &testBasicOptions{},
						},
						"jwt": {
							Enabled: true,
							Score:   20,
							Options: &testJwtOptions{Secret: "secret-secret", Lifetime: 157680000},
						},
					},
					LocalAdminGroupsMap: misc.BoolMap{},
				},
			},
		},
	}

	var loaded testCfg
	err := LoadFile("^test.toml", &loaded)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	err = AddAuthMethod("basic", &testBasicOptions{})
	if err != nil {
		t.Error(err)
	}

	err = AddAuthMethod("jwt", &testJwtOptions{})
	if err != nil {
		t.Error(err)
	}

	err = Check(
		loaded,
		[]any{
			&loaded.HTTP.Listener,
		},
	)
	if err != nil {
		t.Error(err)
	}

	jExpected, err := jsonw.Marshal(expected)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	jLoaded, err := jsonw.Marshal(loaded)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if string(jLoaded) != string(jExpected) {
		t.Errorf("Got: \n%s\nExpected\n%s", jLoaded, jExpected)
	}

	t.Logf("\nSource:\n%s\nResult:\n%s\n", string(GetText()), string(jLoaded))
}

//----------------------------------------------------------------------------------------------------------------------------//

func TestAddMethod(t *testing.T) {
	err := AddAuthMethod("test", &testOptions{})
	if err != nil {
		t.Error(err)
	}
}

type testOptions struct {
	Field1 int     `toml:"field1" mandatory:"true"`
	Field2 int64   `toml:"field2" mandatory:"false"`
	Field3 uint    `toml:"field3" mandatory:"true"`
	Field4 uint64  `toml:"field4"`
	Field5 string  `toml:"field5"`
	Field6 float32 `toml:"field6"`
	Field7 float64 `toml:"field7"`
	Field8 bool    `toml:"field8"`
	//Field9 misc.BoolMap
}

func (options *testOptions) Check(cfg any) (err error) {
	return
}

//----------------------------------------------------------------------------------------------------------------------------//

func TestPreproc(t *testing.T) {
	data := []byte(`auth = { \
endpoints = [], \
					# use /sha?p=<password> for encoding       
# password pwd1       
user1 = "94af3fa5261b347f098bd9cf0fc1c145a20e1f662cb21b0d4a763398ac886f19017cb7d8bfd71df689108511f6f8d0c1ab464a80620d4332379d544ba67131a0", \
	# password pwd2
	user2 = "3ebd594bccb9f9e076cd90eea1f6c46efef9a78a7b58f13fe89d2184160027a3fb6ca1d81073bb64923f3a4fd6b456f04c70a0881827ce43bcfd2642ed93b3d8", \		
}
`)

	expection := `auth = { endpoints = [], user1 = "94af3fa5261b347f098bd9cf0fc1c145a20e1f662cb21b0d4a763398ac886f19017cb7d8bfd71df689108511f6f8d0c1ab464a80620d4332379d544ba67131a0", user2 = "3ebd594bccb9f9e076cd90eea1f6c46efef9a78a7b58f13fe89d2184160027a3fb6ca1d81073bb64923f3a4fd6b456f04c70a0881827ce43bcfd2642ed93b3d8", }`

	data = bytes.TrimSpace(reComment.ReplaceAll(data, []byte{}))
	data = bytes.TrimSpace(reMultiLine.ReplaceAll(data, []byte{' '}))

	if string(data) != string(expection) {
		t.Fatalf("result:\n%q\nis not equal expection:\n%q", data, expection)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

func TestDurationJSON(t *testing.T) {
	data := []struct {
		s string
		e string
		v Duration
	}{
		{
			s: "25",
			e: "25s",
			v: Duration(25 * time.Second),
		},
		{
			s: "15s",
			v: Duration(15 * time.Second),
		},
		{
			s: "2d",
			v: Duration(2 * time.Hour * 24),
		},
		{
			s: "1d2h3m4s5ms6us7ns",
			v: Duration(1*time.Hour*24 + 2*time.Hour + 3*time.Minute + 4*time.Second + 5*time.Millisecond + 6*time.Microsecond + 7*time.Nanosecond),
		},
	}

	type ws struct {
		D Duration
	}

	for i, d := range data {
		w := ws{D: d.v}
		e := d.e
		if d.e == "" {
			e = d.s
		}
		expected := fmt.Sprintf(`{"D":"%s"}`, e)

		s, err := json.Marshal(w)
		if err != nil {
			t.Errorf("[%d] Marshal: %s", i, err)
			continue
		}
		if string(s) != expected {
			t.Errorf("[%d] got '%s', expected '%s'", i, s, expected)
			continue
		}

		w.D = 0

		err = json.Unmarshal(s, &w)
		if err != nil {
			t.Errorf("[%d] Unmarshal: %s", i, err)
			continue
		}
		if w.D != d.v {
			t.Errorf("[%d] got '%d', expected '%d'", i, w.D, d.v)
			continue
		}
	}

}

//----------------------------------------------------------------------------------------------------------------------------//
