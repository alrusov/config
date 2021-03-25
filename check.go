package config

import (
	"reflect"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Common) Check(cfg interface{}) (err error) {
	msgs := misc.NewMessages()

	x.LogBufferDelay, err = misc.Interval2Duration(x.LogBufferDelayS)
	if err != nil {
		msgs.Add("common.log-buffer-delay: %s", err)
	}

	x.MemStatsPeriod, err = misc.Interval2Duration(x.MemStatsPeriodS)
	if err != nil {
		msgs.Add("common.mem-stat-period: %s", err)
	}

	x.LoadAvgPeriod, err = misc.Interval2Duration(x.LoadAvgPeriodS)
	if err != nil {
		msgs.Add("common.load-avg-period: %s", err)
	}

	if x.Name == "" {
		x.Name = misc.AppName()
	}

	if x.LoadAvgPeriod <= 0 {
		x.LoadAvgPeriod = 60
	}

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Listener) Check(cfg interface{}) (err error) {
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

	x.Timeout, err = misc.Interval2Duration(x.TimeoutS)
	if err != nil {
		msgs.Add("listener.timeout: %s", err)
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
func (x *DB) Check(cfg interface{}) (err error) {
	msgs := misc.NewMessages()

	return msgs.Error()
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func Check(cfg interface{}, list []interface{}) error {
	msgs := misc.NewMessages()

	for _, x := range list {
		v := reflect.ValueOf(x)

		if v.Kind() != reflect.Ptr {
			msgs.Add(`"%#v" is not a pointer`, x)
			continue
		}

		m := v.MethodByName("Check")

		if m.Kind() != reflect.Func {
			msgs.Add(`"%#v" doesn't have the "Check" method`, x)
			continue
		}

		e := m.Call([]reflect.Value{reflect.ValueOf(cfg)})

		if len(e) != 1 || e[0].Kind() != reflect.Interface {
			msgs.Add(`"%#v" method "Check" returned illegal value`, x)
			continue
		}

		if e[0].IsNil() {
			continue
		}

		err, ok := e[0].Interface().(error)
		if !ok {
			msgs.Add(`"%#v" method "Check" returned not error value`, x)
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
