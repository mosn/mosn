package javaio

//这个主要是 中间产生的Object
type MiddleClass struct {
	//存储值,这个地方直接存储最终的结果对象.
	self interface{}


	//类全名
	name  string
	//存储元信息
	metaInfo ObjectStreamClass
}
