package config

import (
	"encoding/json"
	"testing"
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

	type cfg struct {
		P0    string                 `toml:"param-0"`
		P1    string                 `toml:"param-1"`
		P2    int                    `toml:"param-2"`
		P3    string                 `toml:"param-3"`
		P4    int                    `toml:"param-4"`
		P5    map[string]interface{} `toml:"param-5"`
		Plast s1                     `toml:"param-last"`
	}

	expected := cfg{
		P0: "***",
		P1: "VAL1",
		P2: 666,
		P3: `!@#$%^&@ qwertyuiop asdfghjkl 123456789 ZZZ`,
		P4: 123456,
		P5: map[string]interface{}{
			"field1": "val1",
			"field2": 777,
		},
		Plast: s1{
			X: 1,
			Y: "Y",
		},
	}

	var loaded cfg
	err := LoadFile("test.toml", &loaded)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	jExpected, err := json.Marshal(expected)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	jLoaded, err := json.Marshal(loaded)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if string(jLoaded) != string(jExpected) {
		t.Errorf("Got: \n%s\nExpected\n%s", jLoaded, jExpected)
	}
}

//----------------------------------------------------------------------------------------------------------------------------//
