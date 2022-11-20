package config

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/naoina/toml"
)

//----------------------------------------------------------------------------------------------------------------------------//

var (
	imapTp = reflect.ValueOf(map[string]any{}).Type()
)

//----------------------------------------------------------------------------------------------------------------------------//

func ConvExtra(src *any, obj any) (err error) {
	if src == nil {
		return fmt.Errorf(`src is nil`)
	}

	if obj == nil {
		return fmt.Errorf(`obj is nil`)
	}

	if reflect.ValueOf(obj).Kind() != reflect.Ptr {
		return fmt.Errorf(`obj is not a pointer`)
	}

	if *src == nil {
		*src = obj
		return
	}

	srcTp := reflect.ValueOf(*src).Type()
	if srcTp != imapTp {
		return fmt.Errorf(`src is "%s", expected "%s"`, srcTp, imapTp)
	}

	buf := new(bytes.Buffer)

	err = toml.NewEncoder(buf).Encode(*src)
	if err != nil {
		return fmt.Errorf(`encode error: %s`, err)
	}

	objTp := reflect.ValueOf(obj).Type()

	err = toml.NewDecoder(buf).Decode(obj)
	if err != nil {
		return fmt.Errorf(`decode error: %s`, err)
	}

	newObjTp := reflect.ValueOf(obj).Type()
	if newObjTp != objTp {
		return fmt.Errorf(`converted data is "%s", expected "%s"`, newObjTp, objTp)
	}

	*src = obj

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
