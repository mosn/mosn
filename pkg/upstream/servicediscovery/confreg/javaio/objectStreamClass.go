package javaio

//这个主要是表达类结构信息
type  ObjectStreamClass struct {
	//类名
	name string
	//序列化 id
	serialVersionUID string
	flags            byte
	fields           []*FieldMetaInfo
	superClass       *ObjectStreamClass
}

func (ser *ObjectStreamClass) SetName(name string) {
	ser.name = name
}

func (ser *ObjectStreamClass) GetName() string {

	return ser.name
}