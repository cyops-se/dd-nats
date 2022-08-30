package types

import (
	"reflect"
)

var typeRegistry = make(map[string]reflect.Type)

func RegisterType(name string, datatype interface{}) {
	// log.Println("Registering type name:", name, reflect.TypeOf(datatype))
	typeRegistry[name] = reflect.TypeOf(datatype)
}

func CreateType(name string) interface{} {
	t := reflect.New(typeRegistry[name]).Interface()
	return t
}

func CreateSlice(name string) interface{} {
	tr := typeRegistry[name]
	if tr == nil {
		return nil
	}

	t := reflect.New(reflect.SliceOf(typeRegistry[name])).Interface()
	return t
}
