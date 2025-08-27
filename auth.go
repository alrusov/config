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
	authMethod struct {
		options any
		check   reflect.Value
	}
)

//----------------------------------------------------------------------------------------------------------------------------//

// AddAuthMethod --
func AddAuthMethod(name string, options any) (err error) {
	_, exists := knownAuthMethods[name]
	if exists {
		return fmt.Errorf(`method "%s" is already defined`, name)
	}

	if options == nil {
		return fmt.Errorf(`options is null`)
	}

	vp := reflect.ValueOf(options)

	if vp.Kind() != reflect.Ptr {
		return fmt.Errorf(`"%#v" is not a pointer`, options)
	}

	v := reflect.Indirect(vp)
	if v.Kind() != reflect.Struct {
		err = fmt.Errorf(`"%#v" is not a pointer to struct`, options)
		return
	}

	m := vp.MethodByName("Check")
	if m.Kind() != reflect.Func {
		return fmt.Errorf(`"%#v" doesn't have the "Check" method`, options)
	}

	method := &authMethod{
		options: options,
		check:   m,
	}

	knownAuthMethods[name] = method

	return
}

//----------------------------------------------------------------------------------------------------------------------------//

// Check --
func (x *Auth) Check(cfg any) (err error) {
	msgs := misc.NewMessages()
	defer msgs.Free()

	x.Endpoints = authSlice2Map(x.EndpointsSlice)

	x.LocalAdminGroupsMap = StringSlice2Map(x.LocalAdminGroups,
		func(name string) string {
			return name
		},
	)

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

		base := fmt.Sprintf(`auth method "%s"`, methodName)

		err = ConvExtra(&method.Options, methodDef.options)
		if err != nil {
			msgs.Add("%s: %s", base, err)
			continue
		}

		if method.Enabled {
			e := methodDef.check.Call([]reflect.Value{reflect.ValueOf(cfg)})
			if len(e) != 1 || e[0].Kind() != reflect.Interface {
				msgs.Add(`%s: Check returned illegal value "#%v"`, base, x)
				continue
			}

			if e[0].IsNil() {
				continue
			}

			err, ok := e[0].Interface().(error)
			if !ok {
				msgs.Add(`%s: Check returned not an error value "#%v"`, base, x)
				continue
			}

			if err != nil {
				msgs.Add(`%s: %s`, base, err)
				continue
			}
		}

	}

	err = msgs.Error()

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
