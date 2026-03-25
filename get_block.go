package config

import (
	"fmt"
	"reflect"
)

//----------------------------------------------------------------------------------------------------------------------------//

// appCfg     -- указатель на конфиг приложения
// blockName  -- название блока (раздела) конфига
// resultPtr  -- указатель, куда будет сохранена ссылка на соответствующий блок, тип указателя (**<тип_блока>) должен соответствовать типу блока

func GetBlock[T any](cfg any, name string) (*T, error) {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return nil, fmt.Errorf("cfg must be non-nil pointer to struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cfg must point to struct")
	}

	f := v.FieldByName(name)
	if !f.IsValid() {
		return nil, fmt.Errorf("field %q not found", name)
	}

	// Приводим к *T независимо от того, T или *T в структуре
	var ptr any

	switch f.Kind() {
	case reflect.Pointer:
		if f.IsNil() {
			return nil, fmt.Errorf("field %q is nil", name)
		}
		ptr = f.Interface()

	case reflect.Struct:
		ptr = f.Addr().Interface()

	default:
		return nil, fmt.Errorf("field %q is not struct or pointer to struct", name)
	}

	res, ok := ptr.(*T)
	if !ok {
		return nil, fmt.Errorf("field %q has incompatible type", name)
	}

	return res, nil
}

//----------------------------------------------------------------------------------------------------------------------------//
