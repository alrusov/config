package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/alrusov/misc"
)

//----------------------------------------------------------------------------------------------------------------------------//

var knownAuthMethods = map[string]*authMethod{}

type (
	// AuthMethodCheck --
	AuthMethodCheck = func(*AuthMethod) (err error)

	authMethod struct {
		pattern  interface{}
		options  map[string]*authMethodOption
		altNames misc.StringMap
		check    AuthMethodCheck
	}

	authMethodOption struct {
		kind reflect.Kind
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

// AddAuthMethod --
func AddAuthMethod(name string, options interface{}, checkConfig AuthMethodCheck) (err error) {
	_, exists := knownAuthMethods[name]
	if exists {
		err = fmt.Errorf(`Method "%s" is already defined`, name)
		return
	}

	if options == nil {
		err = fmt.Errorf(`cfg is null`)
		return
	}

	vp := reflect.ValueOf(options)

	if vp.Kind() != reflect.Ptr {
		err = fmt.Errorf(`"%#v" is not a pointer`, options)
		return
	}

	v := reflect.Indirect(vp)
	if v.Kind() != reflect.Struct {
		err = fmt.Errorf(`"%#v" is not a pointer to struct`, options)
		return
	}

	method := &authMethod{
		pattern:  options,
		options:  map[string]*authMethodOption{},
		altNames: misc.StringMap{},
		check:    checkConfig,
	}

	vt := v.Type()
	nf := v.NumField()

	msgs := misc.NewMessages()

	for i := 0; i < nf; i++ {
		fv := v.Field(i)
		ft := fv.Type()

		field := &authMethodOption{
			kind: fv.Kind(),
		}

		fName := vt.Field(i).Name

		switch field.kind {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64,
			reflect.String:
		default:
			msgs.Add(`Field "%s" has is not a scalar data type (%s)`, fName, ft.String())
			continue
		}

		altName := fName
		tag := vt.Field(i).Tag.Get("toml")
		if tag != "" {
			altName = tag
		}

		method.altNames[altName] = fName
		method.options[vt.Field(i).Name] = field
	}

	err = msgs.Error()
	if err == nil {
		knownAuthMethods[name] = method
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Auth) Check(cfg interface{}) (err error) {
	msgs := misc.NewMessages()

	x.Endpoints = authSlice2Map(x.EndpointsSlice)

	x.Users = make(map[string]User, len(x.UsersMap))

	for u, p := range x.UsersMap {
		u = strings.TrimSpace(u)
		if u == "" {
			msgs.Add(`Empty user name`)
			continue
		}

		g := strings.SplitN(u, "@", 2)
		u = strings.TrimSpace(g[0])

		if len(g) == 1 {
			g = []string{}
		} else {
			g = strings.Split(g[1], ",")
			if len(g) > 0 {
				for i, n := range g {
					g[i] = strings.TrimSpace(n)
					if g[i] == "" {
						msgs.Add(`Empty group for user "%s"`, u)
						continue
					}
				}
			}
		}

		x.Users[u] = User{
			Password: p,
			Groups:   g,
		}
	}

	for methodName, method := range x.Methods {
		methodDef, exists := knownAuthMethods[methodName]
		if !exists {
			msgs.Add(`Unknown auth method "%s"`, methodName)
			continue
		}

		options, err := cloneStruct(methodDef.pattern)
		if err != nil {
			msgs.Add(`%s clone: %v`, methodName, err)
			continue
		}

		method.Options = options.Interface()
		options = reflect.Indirect(options)

		for optName, v := range method.OptionsMap {
			fName, exists := methodDef.altNames[optName]
			if exists {
				optName = fName
			}

			f := options.FieldByName(optName)
			if !f.IsValid() {
				msgs.Add(`%s: unknown field "%s"`, methodName, optName)
				continue
			}

			optDef, exists := methodDef.options[optName]
			if !exists {
				msgs.Add(`%s: no definition for the option "%s" - misconfigured?`, methodName, optName)
				continue
			}

			switch optDef.kind {
			case reflect.Bool:
				vv, err := misc.Iface2Bool(v)
				if err == nil {
					f.SetBool(vv)
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				vv, err := misc.Iface2Int(v)
				if err == nil {
					f.SetInt(vv)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				vv, err := misc.Iface2Uint(v)
				if err == nil {
					f.SetUint(vv)
				}
			case reflect.Float32, reflect.Float64:
				vv, err := misc.Iface2Float(v)
				if err == nil {
					f.SetFloat(vv)
				}
			case reflect.String:
				vv, err := misc.Iface2String(v)
				if err == nil {
					f.SetString(vv)
				}
			default:
				err = fmt.Errorf(`Illegal kind "%s"`, optDef.kind.String())
			}

			if err != nil {
				msgs.Add(`%s.%s: %v`, methodName, optName, err)
				continue
			}
		}

		if methodDef.check != nil && method.Enabled {
			err = methodDef.check(method)
			if err != nil {
				msgs.Add(`%s: %v`, methodName, err)
				continue
			}
		}
	}

	err = msgs.Error()

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

func cloneStruct(src interface{}) (dst reflect.Value, err error) {
	if src == nil {
		err = fmt.Errorf("src is nil")
		return
	}

	vp := reflect.ValueOf(src)

	if vp.Kind() != reflect.Ptr {
		err = fmt.Errorf(`"%#v" is not a pointer`, src)
		return
	}

	if reflect.Indirect(vp).Kind() != reflect.Struct {
		err = fmt.Errorf(`"%#v" is not a pointer to struct`, src)
		return
	}

	dst = reflect.New(reflect.TypeOf(src).Elem())

	srcV := reflect.ValueOf(src).Elem()
	dstV := dst.Elem()
	nf := srcV.NumField()

	for i := 0; i < nf; i++ {
		dstV.Field(i).Set(srcV.Field(i))
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// authSlice2Map --
func authSlice2Map(src map[string][]string) (dst map[string]misc.BoolMap) {
	dst = make(map[string]misc.BoolMap, len(src))

	for path, list := range src {
		path = misc.NormalizeSlashes(path)
		mList := make(misc.BoolMap, len(list))
		for _, u := range list {
			u = strings.TrimSpace(u)
			v := u[0] != '!'
			if !v {
				u = strings.TrimSpace(u[1:])
			}
			if u != "" {
				mList[u] = v
			}
		}
		dst[path] = mList
	}

	return
}

//----------------------------------------------------------------------------------------------------------------------------//
