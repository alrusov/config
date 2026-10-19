package config

import (
	"reflect"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

func GetBlock[T any](cfg any, name string) (res T, err error) {
	v := reflect.ValueOf(cfg)

	res, err = misc.TypedFieldByName[T](v, name, false)
	if err != nil {
		return
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
