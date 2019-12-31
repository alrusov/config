package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Common) Check(cfg interface{}) (err error) {
	var msgs []string

	if x.Name == "" {
		x.Name = misc.AppName()
	}

	if x.LoadAvgPeriod <= 0 {
		x.LoadAvgPeriod = 60
	}

	x.DisabledEndpoints = make(map[string]bool, len(x.DisabledEndpointsSlice))
	for _, name := range x.DisabledEndpointsSlice {
		name = misc.NormalizeSlashes(name)
		x.DisabledEndpoints[name] = true
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Listener) Check(cfg interface{}) (err error) {
	var msgs []string

	x.Addr = strings.TrimSpace(x.Addr)
	x.SSLCombinedPem = strings.TrimSpace(x.SSLCombinedPem)

	if x.Addr == "" {
		if x.SSLCombinedPem == "" {
			x.Addr = ":80"
		} else {
			x.Addr = ":443"
		}
	}

	if x.Timeout == 0 {
		x.Timeout = ListenerDefaultTimeout
	}

	if x.BasicAuthEnabled && len(x.Users) == 0 {
		misc.AddMessage(&msgs, "Listener: basic auth enabled but users list is empty")
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func Check(cfg interface{}, list []interface{}) error {
	var msgs []string

	for _, x := range list {
		v := reflect.ValueOf(x)

		if v.Kind() != reflect.Ptr {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" is not pointer`, x))
			continue
		}

		//if v.Elem().Kind() != reflect.Struct {
		//	misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" is not pointer to struct`, x))
		//	continue
		//}

		m := v.MethodByName("Check")

		if m.Kind() != reflect.Func {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" doesn't have the "Check" method`, x))
			continue
		}

		e := m.Call([]reflect.Value{reflect.ValueOf(cfg)})

		if len(e) != 1 || e[0].Kind() != reflect.Interface {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" method "Check" returned illegal value`, x))
			continue
		}

		if e[0].IsNil() {
			continue
		}

		err, ok := e[0].Interface().(error)
		if !ok {
			misc.AddMessage(&msgs, fmt.Sprintf(`"%#v" method "Check" returned not error value`, x))
			continue
		}

		if err == nil {
			continue
		}

		misc.AddMessage(&msgs, err.Error())
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//
