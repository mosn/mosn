package javaio

import (
	"sync"
)

type CustomSerializeManager struct {
	objects map[string]Serialize
}

var instance *CustomSerializeManager
var once sync.Once

func GetSerializeInstance() *CustomSerializeManager {
	once.Do(func() {
		objects := make(map[string]Serialize, 50)
		instance = &CustomSerializeManager{}
		instance.objects = objects
		instance.AddObjects("java.util.ArrayList", &List{})
	})
	return instance
}

func (customSerializeManager *CustomSerializeManager) GetObjects() map[string]Serialize {

	return customSerializeManager.objects
}

func (customSerializeManager *CustomSerializeManager) GetObject(name string) Serialize {

	return customSerializeManager.objects[name]
}

func (customSerializeManager *CustomSerializeManager) AddObjects(name string, serialize Serialize) {

	customSerializeManager.objects[name] = serialize
}
