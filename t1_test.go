package config

import (
	"reflect"
	"testing"

	"github.com/alrusov/jsonw"
	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

func TestPopulate(t *testing.T) {
	fEnv = func() []string {
		return []string{
			"ENV1=VAL1",
			"ENV2=666",
		}
	}

	type s1 struct {
		X int    `toml:"x"`
		Y string `toml:"y"`
	}

	type http struct {
		Listener Listener `toml:"listener"`
	}

	type basicOptions struct {
	}

	type jwtOptions struct {
		Secret   string `toml:"secret"`
		Lifetime int    `toml:"lifetime"`
	}

	type cfg struct {
		P0    string            `toml:"param-0"`
		P1    string            `toml:"param-1"`
		P2    int               `toml:"param-2"`
		P3    string            `toml:"param-3"`
		P4    int               `toml:"param-4"`
		P5    misc.InterfaceMap `toml:"param-5"`
		Plast s1                `toml:"param-last"`
		HTTP  http              `toml:"http"`
	}

	expected := cfg{
		P0: "***",
		P1: "VAL1",
		P2: 666,
		P3: `!@#$%^&@ qwertyuiop asdfghjkl 123456789 ZZZ`,
		P4: 123456,
		P5: misc.InterfaceMap{
			"field1": "val1",
			"field2": 777,
		},
		Plast: s1{
			X: 1,
			Y: "Y",
		},
		HTTP: http{
			Listener: Listener{
				Addr:                   ":1234",
				SSLCombinedPem:         "",
				Timeout:                6,
				Root:                   "",
				ProxyPrefix:            "/config-test",
				IconFile:               "/tmp/favicon.ico",
				DisabledEndpointsSlice: []string{"/aaa*", "!/aaa/bbb"},
				DisabledEndpoints:      misc.BoolMap{"/aaa*": true, "!/aaa/bbb": true},
				Auth: Auth{
					EndpointsSlice: []string{"/xxx"},
					Endpoints:      misc.BoolMap{"/xxx": true},
					Users:          misc.StringMap{"test-user1": "pwd1", "test-user2": "pwd2"},
					Methods: map[string]*AuthMethod{
						"basic": &AuthMethod{
							Enabled:    true,
							OptionsMap: misc.InterfaceMap{},
							Options:    &basicOptions{},
						},
						"jwt": &AuthMethod{
							Enabled:    true,
							OptionsMap: misc.InterfaceMap{"secret": "secret-secret", "lifetime": float64(157680000)},
							Options:    &jwtOptions{Secret: "secret-secret", Lifetime: 157680000},
						},
					},
				},
			},
		},
	}

	var loaded cfg
	err := LoadFile("^test.toml", &loaded)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	err = AddAuthMethod("basic", &basicOptions{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = AddAuthMethod("jwt", &jwtOptions{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = Check(
		loaded,
		[]interface{}{
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
	type cfg struct {
		Field1 int     `toml:"field1" mandatory:"true"`
		Field2 int64   `toml:"field2" mandatory:"false"`
		Field3 uint    `toml:"field3" mandatory:"true"`
		Field4 uint64  `toml:"field4"`
		Field5 string  `toml:"field5"`
		Field6 float32 `toml:"field6"`
		Field7 float64 `toml:"field7"`
		Field8 bool    `toml:"field8"`
		//Field9 map[string]bool
	}

	err := AddAuthMethod("test", &cfg{},
		func(cfg *AuthMethod) (err error) {
			return
		},
	)
	if err != nil {
		t.Error(err)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//

func TestCloneStruct(t *testing.T) {
	type data struct {
		Field1 int     `toml:"field1" mandatory:"true"`
		Field2 int64   `toml:"field2" mandatory:"false"`
		Field3 uint    `toml:"field3" mandatory:"true"`
		Field4 uint64  `toml:"field4"`
		Field5 string  `toml:"field5"`
		Field6 float32 `toml:"field6"`
		Field7 float64 `toml:"field7"`
		Field8 bool    `toml:"field8"`
	}
	src := &data{1, 2, 3, 4, "qwerty", 1.1, 2.2, true}

	dstV, err := cloneStruct(src)
	if err != nil {
		t.Fatal(err)
	}

	dst := dstV.Interface()

	if !reflect.DeepEqual(src, dst) {
		t.Fatalf("src=%#v\nnot equal\ndst=%#v", src, dst)
	}
}
