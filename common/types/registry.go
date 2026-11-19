package types

import (
	"reflect"
	"sync"
)

var typeRegistry = make(map[string]reflect.Type)
var typeRegistryLock sync.RWMutex

func RegisterType(name string, datatype interface{}) {
	t := reflect.TypeOf(datatype)
	if t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeRegistryLock.Lock()
	defer typeRegistryLock.Unlock()
	typeRegistry[name] = t
}

func CreateType(name string) interface{} {
	typeRegistryLock.RLock()
	tr := typeRegistry[name]
	typeRegistryLock.RUnlock()

	if tr == nil {
		return nil
	}

	return reflect.New(tr).Interface()
}

func CreateSlice(name string) interface{} {
	typeRegistryLock.RLock()
	tr := typeRegistry[name]
	typeRegistryLock.RUnlock()

	if tr == nil {
		return nil
	}

	return reflect.New(reflect.SliceOf(tr)).Interface()
}
