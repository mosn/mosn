package javaio

import (
	"bytes"
	"encoding/binary"
	"regexp"
	"strconv"
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type OutputObjectStream struct {
	out  *bytes.Buffer
	refs []interface{}
}

func (output *OutputObjectStream) Init() {

	output.refs = make([]interface{}, 0)

	contents := make([]byte, 1<<20)
	output.out = bytes.NewBuffer(contents)
}

/**
write header
 */
func (output *OutputObjectStream) writeStreamHeader() {

	binary.BigEndian.PutUint16(output.out.Bytes(), STREAM_MAGIC)
	binary.BigEndian.PutUint16(output.out.Bytes(), STREAM_VERSION)

}

/**
write content
 */
func (output *OutputObjectStream) writeContent(obj *MiddleClass) []byte {

	output.writeObject(obj)

	return output.out.Bytes()

}

/**
write single object
  // object:
  //   newObject
  //   newClass
  //   newArray
  //   newString
  //   newEnum
  //   newClassDesc
  //   prevObject
  //   nullReference
  //   exception
  //   TC_RESET

 */
func (output *OutputObjectStream) writeObject(obj *MiddleClass) {

	var handle int32

	if obj.self == nil {
		output.writeNull()
	} else if output.lookupHandle(obj) != -1 {
		handle = output.lookupHandle(obj)
		output.writeHandle(handle)
	} else if str, ok := obj.self.(string); ok {
		output.writeString(str)
	} else if output.isEnum(obj) {
		output.writeEnum(obj)
	} else if output.isArray(obj) {
		output.writeArray(obj)
	} else {
		output.writeOrdinaryObject(obj)
	}

}

/**
write null
 */
func (output *OutputObjectStream) writeNull() {
	output.out.WriteByte(TC_NULL)

}
func (output *OutputObjectStream) lookupHandle(obj interface{}) int32 {

	return output.indexOf(obj, output.refs)
}

/**
引用 handle
 */
func (output *OutputObjectStream) writeHandle(i int32) {

	output.out.WriteByte(TC_REFERENCE)
	output.putInt32(i)
}

func (output *OutputObjectStream) putBoolean(i bool) {

	if i {
		output.out.WriteByte(1)
	}else{
		output.out.WriteByte(0)
	}
}

func (output *OutputObjectStream) putInt32(i int32) {

	binary.BigEndian.PutUint32(output.out.Bytes(), uint32(i))
}

func (output *OutputObjectStream) putInt16(i int16) {

	binary.BigEndian.PutUint16(output.out.Bytes(), uint16(i))
}

func (output *OutputObjectStream) putInt64(i int64) {

	binary.BigEndian.PutUint64(output.out.Bytes(), uint64(i))
}

func (output *OutputObjectStream) putFloat64(i float64) {

	binary.BigEndian.PutUint64(output.out.Bytes(), uint64(i))
}

func (output *OutputObjectStream) putFloat32(i float32) {

	binary.BigEndian.PutUint32(output.out.Bytes(), uint32(i))
}

func (output *OutputObjectStream) writeString(str string) {

	output.newHandle(str)
	b := []byte(str)
	length := len(b)

	if length <= 0xffff {
		output.out.WriteByte(TC_STRING)

		binary.BigEndian.PutUint16(output.out.Bytes(), uint16(length))
	} else {
		output.out.WriteByte(TC_LONGSTRING)

		output.putInt64(int64(length))
	}

	output.out.Write(b)

}

func (output *OutputObjectStream) newHandle(i interface{}) {

	if output.refs == nil {
		output.refs = make([]interface{}, 1)
		output.refs[0] = i

	} else {
		/*tmp := make([]interface{}, len(input.refs)+1)
		copy(tmp, input.refs)
		tmp[len(input.refs)] = i
		input.refs = tmp*/
		output.refs = append(output.refs, i)
	}

}

func (output *OutputObjectStream) isEnum(obj *MiddleClass) bool {

	var metaInfo = obj.metaInfo

	for metaInfo.superClass != nil {

		if metaInfo.name == "java.lang.Enum" {
			return true
		}

		metaInfo = *metaInfo.superClass
	}
	return false
}
func (output *OutputObjectStream) writeBlockHeader(length int32) {
	if (length <= 0xFF) {
		// Data blocks shorter than 256 bytes are prefixed with a 2-byte header;
		p := make([]byte, 2)
		p[0] = TC_BLOCKDATA
		p[1] = byte(length)
		output.out.Write(p)
	} else {
		// All others start with a 5-byte header.
		// TC_BLOCKDATALONG
		output.out.WriteByte(TC_BLOCKDATALONG)
		output.putInt32(length)
	}
}

func (output *OutputObjectStream) writeEnum(obj *MiddleClass) {

}
func (output *OutputObjectStream) isArray(obj *MiddleClass) bool {

	array := obj.metaInfo.name == "["

	return array
}
func (output *OutputObjectStream) writeArray(obj *MiddleClass) {

	output.out.WriteByte(TC_ARRAY)

	output.writeClassDesc(&obj.metaInfo)

	output.newHandle(obj)

	value, ok := obj.self.([]interface{})

	if ok {

		length := len(value)
		output.putInt32(int32(length))

		if output.isElementPrimitive(obj) {

		}

	} else {
		log.DefaultLogger.Errorf("value is not an array.should check")
	}

}
func (output *OutputObjectStream) writeOrdinaryObject(obj *MiddleClass) {

	output.out.WriteByte(TC_OBJECT)

	output.writeClassDesc(&obj.metaInfo)

	output.newHandle(obj)

	output.writeSerializeData(obj)
}

func (output *OutputObjectStream) indexOf(element interface{}, data []interface{}) int32 {
	for k, v := range data {
		if element == v {
			return int32(k)
		}
	}
	return -1 //not found.
}

/**
写类定义
 */
func (output *OutputObjectStream) writeClassDesc(obj *ObjectStreamClass) {

	log.DefaultLogger.Infof("start to write classDesc,desc=%+v", obj)

	var handle int32

	if obj == nil {
		output.writeNull()
	} else if output.lookupHandle(obj) != -1 {
		handle = output.lookupHandle(obj)
		output.writeHandle(handle)
	} else {
		output.writeNonProxyDesc(obj)
	}

	log.DefaultLogger.Infof("end to write classDesc,desc=%+v", obj)

}
func (output *OutputObjectStream) isElementPrimitive(obj *MiddleClass) bool {

	var descName = obj.metaInfo.name

	return output.isPrimitive(descName[1:2])

}

/**
判断是否原生类型.
 */
func (output *OutputObjectStream) isPrimitive(str string) bool {

	pattern := "BCDFIJSZ"

	match, err := regexp.MatchString(pattern, str)

	if err == nil {
		return match
	} else {
		log.DefaultLogger.Errorf("some error occuurs,err=%+v", err)
		return false
	}

}

/**
对应_writeSerialData
 */
func (output *OutputObjectStream) writeSerializeData(obj *MiddleClass) {
	var classDesc = obj.metaInfo
	var flags = classDesc.flags
	var classname = classDesc.name
	var superClass = classDesc.superClass

	for superClass != nil {
		flags |= superClass.flags
		superClass = superClass.superClass
	}

	if flags&SC_SERIALIZABLE > 0 {
		var customObject = GetSerializeInstance().GetObject(classname)

		if customObject == nil {
			log.DefaultLogger.Errorf("need to be add to custom serializeInstance", classname)
		}

		var hasWriteObjectMethod = customObject != nil

		if flags&SC_WRITE_METHOD > 0 {

			if hasWriteObjectMethod {

				customObject.WriteObject(output, obj)
				output.out.WriteByte(TC_ENDBLOCKDATA)
			} else {
				//抛异常
				log.DefaultLogger.Errorf("impossible has no write method")
			}
		} else {
			if hasWriteObjectMethod {

				customObject.WriteObject(output, obj)

				output.out.WriteByte(TC_ENDBLOCKDATA)

			} else {
				output.defaultWriteFields(obj)
			}
		}

	} else if flags&SC_EXTERNALIZABLE > 0 {
		if flags&SC_BLOCK_DATA > 0 {

			log.DefaultLogger.Errorf("Not implement writeObjectAnnotation()")

		} else {
			log.DefaultLogger.Errorf("Not implement writeExternalContents()")
		}
	} else {
		//理论上不能走到这里
		log.DefaultLogger.Errorf("Illegal flags: ", flags)
	}
}
func (output *OutputObjectStream) defaultWriteFields(obj *MiddleClass) {

	var fieldDesc = concatFields(obj.metaInfo)

	for i := 0; i < len(fieldDesc); i++ {
		fieldMetaInfo := fieldDesc[i]
		var fieldType = fieldMetaInfo.filedType

		reflectType := reflect.TypeOf(obj.self).Elem()
		log.DefaultLogger.Infof("new reflect type,%+s", reflectType)

		currentObj := reflect.New(reflectType)

		method := currentObj.MethodByName("Get" + fieldMetaInfo.name)

		v := make([]reflect.Value, 0)

		values := method.Call(v)

		var value = values[0].Interface()
		if output.isPrimitive(fieldType) {
			output.writePrimitive(fieldType, value)
		} else {
			valueClass := MiddleClass{}
			valueClass.self = value
			infoInterface, ok := value.(MetaInfoInterface)

			if ok {
				javaMetaInfo := infoInterface.ConstructJavaMetaInfo()
				valueClass.metaInfo = *javaMetaInfo
				output.writeObject(&valueClass)
			} else {
				log.DefaultLogger.Errorf("write field occurs error,obj=%+v", obj)
			}
		}

	}

}
func (output *OutputObjectStream) writeNonProxyDesc(desc *ObjectStreamClass) {

	output.out.WriteByte(TC_CLASSDESC)

	output.newHandle(desc)

	output.writeUTF(desc.name)

	i, err := strconv.ParseInt(desc.serialVersionUID, 10, 64)

	if err == nil {
		output.putInt64(i)

	} else {
		log.DefaultLogger.Errorf("it is impossile,convert serialVersionUID error")
	}

	output.writeClassDescInfo(desc)

}
func (output *OutputObjectStream) writeUTF(str string) {

	b := []byte(str)
	length := len(b)

	if length <= 0xffff {
		binary.BigEndian.PutUint16(output.out.Bytes(), uint16(length))
		output.out.Write(b)
	} else {
		log.DefaultLogger.Errorf("it is impossile, writeUTF error")
	}

}
func (output *OutputObjectStream) writeClassDescInfo(desc *ObjectStreamClass) {

	output.out.WriteByte(desc.flags)

	length := len(desc.fields)
	binary.BigEndian.PutUint16(output.out.Bytes(), uint16(length))

	for i := 0; i < length; i++ {

		fieldMetaInfo := desc.fields[i]
		output.putChar(fieldMetaInfo.filedType)

		if !output.isPrimitive(fieldMetaInfo.filedType) {
			output.writeTypeString(fieldMetaInfo.className)
		}
	}

	output.writeClassAnnotation(desc)

	output.writeClassDesc(desc.superClass)

}

func (output *OutputObjectStream) putChar(str string) {
	output.out.WriteByte(str[0])
}
func (output *OutputObjectStream) writeTypeString(str string) {

	var handle int32

	if str == "" {
		output.writeNull()
	} else if output.lookupHandle(str) != -1 {
		handle = output.lookupHandle(str)
		output.writeHandle(handle)
	} else {
		output.writeString(str)
	}
}
func (output *OutputObjectStream) writeClassAnnotation(desc *ObjectStreamClass) {
	output.out.WriteByte(TC_ENDBLOCKDATA)
}
func (output *OutputObjectStream) writePrimitive(fieldType string, value interface{}) {
	if fieldType == "B" {
		output.out.WriteByte(value.(byte))
	} else if fieldType == "C" {
		binary.BigEndian.PutUint16(output.out.Bytes(), value.(uint16))
	} else if fieldType == "D" {
		output.putFloat64(value.(float64))
	} else if fieldType == "F" {
		output.putFloat32(value.(float32))
	} else if fieldType == "I" {
		output.putInt32(value.(int32))
	} else if fieldType == "J" {
		output.putInt64(int64(value.(int64)))
	} else if fieldType == "S" {
		output.putInt16(value.(int16))

	} else if fieldType == "Z" {

		var b bool
		b = value.(bool)
		if b == true {
			output.out.WriteByte(1)
		} else {
			output.out.WriteByte(0)
		}
	} else {
		log.DefaultLogger.Errorf("Illegal primitive type:,%+s", fieldType)
	}
}
