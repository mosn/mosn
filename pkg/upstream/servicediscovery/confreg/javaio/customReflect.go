package javaio

import "sync"

type CustomReflectManager struct {
	objects map[string]interface{}
}

var customReflectManager *CustomReflectManager
var once1 sync.Once

func GetReflectInstance() *CustomReflectManager {
	once1.Do(func() {
		objects := make(map[string]interface{}, 50)
		customReflectManager = &CustomReflectManager{}
		customReflectManager.objects = objects
		customReflectManager.AddObjects("java.util.ArrayList", &List{})

	})
	return customReflectManager
}

func (customSerializeManager *CustomReflectManager) AddObjects(name string, serialize interface{}) {

	customSerializeManager.objects[name] = serialize
}

func (customSerializeManager *CustomReflectManager) GetObject(name string) interface{} {

	return customSerializeManager.objects[name]
}
