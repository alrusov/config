package config

import (
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

	if x.Timeout <= 0 {
		x.Timeout = ListenerDefaultTimeout
	}

	if x.JWTlifetime <= 0 {
		x.JWTlifetime = JWTdefaultLifetime
	}

	if x.BasicAuthEnabled && len(x.Users) == 0 {
		misc.AddMessage(&msgs, "Listener: basic auth enabled but users list is empty")
	}

	if x.JWTsecret != "" && len(x.Users) == 0 {
		misc.AddMessage(&msgs, "Listener: jwt auth enabled but users list is empty")
	}

	if x.Root != "" {
		x.Root, err = misc.AbsPath(x.Root)
		if err != nil {
			misc.AddMessage(&msgs, "Listener.Root: %s", err.Error())
		}
	}

	x.DisabledEndpoints = Slice2Map(x.DisabledEndpointsSlice,
		func(name string) string {
			return misc.NormalizeSlashes(name)
		},
	)

	x.JWTEndpoints = Slice2Map(x.JWTEndpointsSlice,
		func(name string) string {
			return misc.NormalizeSlashes(name)
		},
	)

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func Check(cfg interface{}, list []interface{}) error {
	var msgs []string

	for _, x := range list {
		v := reflect.ValueOf(x)

		if v.Kind() != reflect.Ptr {
			misc.AddMessage(&msgs, `"%#v" is not a pointer`, x)
			continue
		}

		//if v.Elem().Kind() != reflect.Struct {
		//	misc.AddMessage(&msgs, `"%#v" is not a pointer to struct`, x)
		//	continue
		//}

		m := v.MethodByName("Check")

		if m.Kind() != reflect.Func {
			misc.AddMessage(&msgs, `"%#v" doesn't have the "Check" method`, x)
			continue
		}

		e := m.Call([]reflect.Value{reflect.ValueOf(cfg)})

		if len(e) != 1 || e[0].Kind() != reflect.Interface {
			misc.AddMessage(&msgs, `"%#v" method "Check" returned illegal value`, x)
			continue
		}

		if e[0].IsNil() {
			continue
		}

		err, ok := e[0].Interface().(error)
		if !ok {
			misc.AddMessage(&msgs, `"%#v" method "Check" returned not error value`, x)
			continue
		}

		if err == nil {
			continue
		}

		misc.AddMessage(&msgs, "%s", err.Error())
	}

	return misc.JoinedError(msgs)
}

//----------------------------------------------------------------------------------------------------------------------------//

// Slice2Map --
func Slice2Map(src []string, conv func(string) string) (dst map[string]bool) {
	dst = make(map[string]bool, len(src))

	for _, name := range src {
		if conv != nil {
			name = conv(name)
		}
		dst[name] = true
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
