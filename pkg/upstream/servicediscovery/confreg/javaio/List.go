package javaio

import (
	log "alipay.com/sofa-mesh/log"
)

type List struct {
	value []string
	size  int32
}

func (hashMap *List) ConstructJavaMetaInfo() *ObjectStreamClass {

	var metaInfo = ObjectStreamClass{}

	metaInfo.name = "java.util.HashMap"
	metaInfo.serialVersionUID = "362498820763181265"
	metaInfo.flags = 3
	metaInfo.fields = make([]*FieldMetaInfo, 0)
	metaInfo.fields = append(metaInfo.fields, &FieldMetaInfo{filedType: "F", name: "loadFactor"})
	metaInfo.fields = append(metaInfo.fields, &FieldMetaInfo{filedType: "I", name: "threshold"})
	metaInfo.superClass = nil
	return &metaInfo
}

func (hashMap *List) WriteObject(stream *OutputObjectStream, obj *MiddleClass) {

	log.Info("start HashMap writeObject")

	log.Info("end HashMap writeObject")

}

func (hashMap *List) ReadObject(input *InputObjectStream, obj *MiddleClass) {

	if obj.self == nil {
		i := List{}
		_v := make([]string, 0)
		i.value = _v
		obj.self = &i
	}

	selfList, ok := obj.self.(*List)

	if ok {
		input.defaultReadFields(obj)

		selfList = obj.self.(*List)

		input.readBlockHeader()
		input.readInt32(input.in)
		size := selfList.size

		for i := 0; i < int(size); i++ {
			content := input.ReadContent()

			if str, ok := content.(string); ok {
				selfList.value = append(selfList.value, str)
			} else {
				panic("now,we only support list<string>")
			}
		}

		obj.self = selfList

	} else {
		panic("now,we only support list<string>")
	}

}

//for serialize
func (hashMap *List) Setsize(size int32) {
	hashMap.size = size
}

//for serialize
func (hashMap *List) Getsize() int32 {
	return hashMap.size
}

//for serialize
func (hashMap *List) GetValue() []string {
	return hashMap.value
}
