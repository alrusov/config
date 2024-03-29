package config

import (
	"reflect"
	"strings"
	"time"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Common) Check(cfg any) (err error) {
	msgs := misc.NewMessages()

	if x.Name == "" {
		x.Name = misc.AppName()
	}

	if x.LoadAvgPeriod <= 0 {
		x.LoadAvgPeriod = Duration(60 * time.Second)
	}

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Listener) Check(cfg any) (err error) {
	msgs := misc.NewMessages()

	x.Addr = strings.TrimSpace(x.Addr)

	if x.Addr == "" {
		if x.SSLCombinedPem == "" {
			x.Addr = ":80"
		} else {
			x.Addr = ":443"
		}
	}

	if x.Root != "" {
		x.Root, err = misc.AbsPath(x.Root)
		if err != nil {
			msgs.Add("listener.root: %s", err)
		}
	}

	if x.ProxyPrefix != "" {
		x.ProxyPrefix = misc.NormalizeSlashes("/" + x.ProxyPrefix)
	}

	if x.SSLCombinedPem != "" {
		x.SSLCombinedPem, err = misc.AbsPath(x.SSLCombinedPem)
		if err != nil {
			msgs.Add("listener.ssl-combined-pem: %s", err)
		}
	}

	if x.Timeout <= 0 {
		x.Timeout = ListenerDefaultTimeout
	}

	if x.IconFile != "" {
		x.IconFile, err = misc.AbsPath(x.IconFile)
		if err != nil {
			msgs.Add("listener.icon-file: %s", err)
		}
	}

	x.DisabledEndpoints = StringSlice2Map(x.DisabledEndpointsSlice,
		func(name string) string {
			return misc.NormalizeSlashes(name)
		},
	)

	err = x.Auth.Check(cfg)
	if err != nil {
		msgs.Add("listener.auth: %s", err)
	}

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//
// Check --
func (x *DB) Check(cfg any) (err error) {
	msgs := misc.NewMessages()

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func Check(cfg any, list []any) error {
	msgs := misc.NewMessages()

	for _, x := range list {
		v := reflect.ValueOf(x)

		if v.Kind() != reflect.Ptr {
			msgs.Add(`"%#v" is not a pointer`, x)
			continue
		}

		m := v.MethodByName("Check")

		if m.Kind() != reflect.Func {
			msgs.Add(`"%#v" doesn't have the Check function`, x)
			continue
		}

		e := m.Call([]reflect.Value{reflect.ValueOf(cfg)})

		if len(e) != 1 || e[0].Kind() != reflect.Interface {
			msgs.Add(`"%#v" Check function returned an illegal value`, x)
			continue
		}

		if e[0].IsNil() {
			continue
		}

		err, ok := e[0].Interface().(error)
		if !ok {
			msgs.Add(`"%#v" Check function returned not error type value`, x)
			continue
		}

		if err == nil {
			continue
		}

		msgs.Add("%s", err)
	}

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//

// StringSlice2Map --
func StringSlice2Map(src []string, conv func(name string) string) (dst misc.BoolMap) {
	dst = make(misc.BoolMap, len(src))

	for _, name := range src {
		if conv != nil {
			name = conv(name)
		}
		dst[name] = true
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
