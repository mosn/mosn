package javaio

import (
	"bytes"
	"strings"
	"encoding/binary"
	"strconv"
	"reflect"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type InputObjectStream struct {
	withType string //TODO 这里改成bool
	in       *bytes.Buffer
	refs     []interface{}
}

func (input *InputObjectStream) SetWithType(withType string) {
	input.withType = withType
}

//这里直接读了 header
func (input *InputObjectStream) SetBytes(contents []byte) {
	input.in = bytes.NewBuffer(contents)

	input.ReadHeader()
}

func (input *InputObjectStream) ReadContent() interface{} {

	la := input.lookAhead()

	if (la == TC_BLOCKDATA) {
		return input.readBlockDataShort()
	} else if (la == TC_BLOCKDATALONG) {
		return input.readBlockDataLong()
	} else {
		return input.readObject()
	}

}

func (input *InputObjectStream) ReadHeader() {
	// stream:
	//   magic version contents
	var magic = input.readUInt16(input.in)
	var version = input.readUInt16(input.in)
	if (magic != STREAM_MAGIC || version != STREAM_VERSION) {

		log.DefaultLogger.Fatal("error format")
	}
}

//读一个 byte
func (input *InputObjectStream) lookAhead() byte {
	tempbytes := make([]byte, 1)
	_, err := input.peek(input.in, tempbytes)
	if err != nil {
		//FIXME 这里抛异常

		return '0'
	} else {
		return tempbytes[0]
	}
}

func (input *InputObjectStream) peek(buf *bytes.Buffer, b []byte) (int, error) {
	buf2 := bytes.NewBuffer(buf.Bytes())
	return buf2.Read(b)
}

func (input *InputObjectStream) readBlockHeader() interface{} {

	var flag, _ = input.in.ReadByte()
	if flag == TC_BLOCKDATA {
		// 0x77 size(1 byte)

		var data, _ = input.in.ReadByte()
		return data
	} else if flag == TC_BLOCKDATALONG {
		// 0x7A size(4 bytes)
		// 返回后面可以读取的字节数.
		var size = input.readInt(input.in)
		return size
	} else {

		log.DefaultLogger.Errorf("Illegal lookahead,%d", flag)

		return nil
	}

}

func (input *InputObjectStream) readBlockDataShort() interface{} {

	//赞不支持
	return nil

}

func (input *InputObjectStream) readBlockDataLong() interface{} {

	//暂不支持
	return nil
}

func (input *InputObjectStream) readObject() interface{} {
	la := input.lookAhead()
	if la == TC_OBJECT {
		return input.readNewObject()
	} else if la == TC_CLASS {
		return input.readNewClass()
	} else if la == TC_ARRAY {
		return input.readNewArray()
	} else if la == TC_STRING || la == TC_LONGSTRING {
		return input.readNewString()
	} else if la == TC_ENUM {
		return input.readNewEnum()
	} else if la == TC_CLASSDESC || la == TC_PROXYCLASSDESC {
		return input.readNewClassDesc()
	} else if la == TC_REFERENCE {
		var obj = input.readPrevObject()
		return input.filter(obj, input.withType)
	} else if la == TC_NULL {
		return input.readNull()
	} else if la == TC_EXCEPTION {
		return input.readException()
	} else if la == TC_RESET {
		return input.readReset()
	} else {
		//打印日志
		log.DefaultLogger.Errorf("some byte can not be processed,%s", la)
		return nil
	}

}
func (input *InputObjectStream) filter(obj interface{}, withType string) interface{} {

	obj2, ok := obj.(MiddleClass)

	if !ok {

		return obj
	} else {
		var class = obj2.metaInfo;
		var fields = concatFields(class);
		var index = strings.Index(class.name, "java.lang.")

		var isPrimitive = (index == 0 && len(fields) == 1 && fields[0].name == "value")

		if isPrimitive {
			return obj2.self
		} else {
			return obj2.self
		}
	}

}

/***
合并 field
 */
func concatFields(serializeClass ObjectStreamClass) []*FieldMetaInfo {

	if serializeClass.superClass == nil {

		return serializeClass.fields

	} else {
		var parent = concatFields(*serializeClass.superClass)
		return merge(parent, serializeClass.fields)
	}
}

func merge(x []*FieldMetaInfo, y []*FieldMetaInfo) []*FieldMetaInfo {

	xlen := len(x) //x数组的长度
	ylen := len(y) //y数组的长度
	length := xlen + ylen
	z := make([]*FieldMetaInfo, length) //创建一个大小为xlen+ylen的数组切片
	for i := 0; i < xlen; i++ {
		z[i] = x[i]
	}

	for i := 0; i < ylen; i++ {
		z[xlen+i] = y[i]
	}

	return z

}
func (input *InputObjectStream) readNewObject() interface{} {

	input.in.ReadByte()

	var obj = MiddleClass{self: nil}
	obj.metaInfo = *input.readClassDesc()

	var size int
	if input.refs == nil {
		size = 0
	} else {
		size = len(input.refs)
	}

	input.newHandle(obj)
	input.readClassData(&obj)

	input.refs[size] = obj

	var ret = input.filter(obj, input.withType)
	return ret
}
func (input *InputObjectStream) readNewClass() interface{} {

	//不实现
	return nil
}
func (input *InputObjectStream) readNewArray() interface{} {
	input.in.ReadByte()

	var obj = MiddleClass{}

	obj.metaInfo = *input.readClassDesc()

	input.newHandle(obj)

	input.readArrayItems(obj)

	if obj.metaInfo.name == "[B" {
		//obj.self = new Buffer(obj.$);
		//先不做
	}

	if input.withType != "" {

		return obj

	} else {
		return obj.self
	}
}
func (input *InputObjectStream) readNewString() interface{} {
	byteType, err := input.in.ReadByte()

	if err != nil {
		//FIXME

		return nil
	} else {
		var str = input.readUTFString(byteType == TC_LONGSTRING)
		input.newHandle(str)

		return str
	}

}
func (input *InputObjectStream) readNewEnum() interface{} {

	log.DefaultLogger.Errorf("impossible")
	return nil
}

//发现新 class 描述信息,从这里读取
func (input *InputObjectStream) readNewClassDesc() interface{} {
	var la = input.lookAhead()

	if la == TC_CLASSDESC {
		return input.readNonProxyDesc()
	} else if la == TC_PROXYCLASSDESC {
		//throw new Error('Not implement _readNewClassDesc.PROXYCLASSDESC');

		//panic FIXMe

		return nil
	} else {
		//throw new Error('Illegal lookahead: 0x' + la.toString(16));
		//panic FIXMe

		return nil
	}
}
func (input *InputObjectStream) readPrevObject() interface{} {
	// prevObject
	//   TC_REFERENCE (int)handle
	input.in.ReadByte()
	var id = input.readInt(input.in)
	var obj = input.refs[id-baseWireHandle]
	return obj
}
func (input *InputObjectStream) readNull() interface{} {
	input.in.ReadByte()
	return nil

}
func (input *InputObjectStream) readException() interface{} {

	//不实现

	return nil
}
func (input *InputObjectStream) readReset() interface{} {
	//不实现

	return nil
}
func (input *InputObjectStream) readClassDesc() *ObjectStreamClass {
	var la = input.lookAhead()
	if (la == TC_CLASSDESC) {
		streamClass := input.readNewClassDesc().(ObjectStreamClass)
		return &streamClass
	} else if (la == TC_REFERENCE) {
		streamClass := input.readPrevObject().(ObjectStreamClass)
		return &streamClass
	} else if (la == TC_NULL) {
		input.in.ReadByte()
		//TODO 这里要改下
		return nil
	} else {

		//FIXMe panic
		return nil
	}
}
func (input *InputObjectStream) newHandle(i interface{}) {

	if input.refs == nil {
		input.refs = make([]interface{}, 1)
		input.refs[0] = i

		log.DefaultLogger.Infof("> _newHandle | index = %d, obj = %+v", len(input.refs), i)
	} else {
		/*tmp := make([]interface{}, len(input.refs)+1)
		copy(tmp, input.refs)
		tmp[len(input.refs)] = i
		input.refs = tmp*/
		log.DefaultLogger.Infof("> _newHandle | index = %d, obj = %+v", len(input.refs), i)
		input.refs = append(input.refs, i)
	}

}

/**
读取 classData
 */
func (input *InputObjectStream) readClassData(obj *MiddleClass) {

	// classdata:
	//   nowrclass                 // SC_SERIALIZABLE & classDescFlag &&
	//                             // !(SC_WRITE_METHOD & classDescFlags)
	//   wrclass objectAnnotation  // SC_SERIALIZABLE & classDescFlag &&
	//                             // SC_WRITE_METHOD & classDescFlags
	//   externalContents          // SC_EXTERNALIZABLE & classDescFlag &&
	//                             // !(SC_BLOCKDATA  & classDescFlags
	//   objectAnnotation          // SC_EXTERNALIZABLE & classDescFlag&&
	//                             // SC_BLOCKDATA & classDescFlags

	var classDesc = obj.metaInfo
	var flags = classDesc.flags
	var classname = classDesc.name
	var superClass = classDesc.superClass

	//TODO 这里我加的
	obj.name = classname
	// try to detect class have readObject or not, (its own method or inherited from superClass)
	for superClass != nil {
		flags |= superClass.flags
		superClass = superClass.superClass
	}
	if flags&SC_SERIALIZABLE > 0 {
		var customObject = GetSerializeInstance().GetObject(classname)

		if customObject == nil {
			log.DefaultLogger.Errorf("need to be add to custom serializeInstance", classname)
		}

		var hasReadObjectMethod = customObject != nil

		if flags&SC_WRITE_METHOD > 0 {

			if hasReadObjectMethod {
				customObject.ReadObject(input, obj)
				input.in.ReadByte()
			} else {
				//抛异常
				log.DefaultLogger.Errorf("impossible has no write method")
			}
		} else {
			if hasReadObjectMethod {

				customObject.ReadObject(input, obj)

			} else {
				input.ReadNowrclass(obj)
			}
		}

	} else if flags&SC_EXTERNALIZABLE > 0 {
		if flags&SC_BLOCK_DATA > 0 {
			input.ReadObjectAnnotation(*obj)
		} else {
			input.readExternalContents()
		}
	} else {
		//理论上不能走到这里
	}

}
func (input *InputObjectStream) ReadNowrclass(obj *MiddleClass) {
	input.defaultReadFields(obj)

}
func (input *InputObjectStream) ReadObjectAnnotation(class MiddleClass) {

}
func (input *InputObjectStream) readExternalContents() {

}
func (input *InputObjectStream) readArrayItems(obj MiddleClass) {
	/*// (int)<size> values[size]
	var size = input.readInt(bytes.NewBuffer(input.in.Bytes()))

	// values:        // The size and types are described by the
	//                // classDesc for the current object
	//var classType = obj.metaInfo.name[1]
	//FIXMe 数组功能
	var classType = obj.metaInfo.name
	// [I

	selfLength := len(obj.self)
	t := make([]interface{}, selfLength+size)
	copy(t, obj.self)

	for i := 0; i < size; i++ {
		t[selfLength+i:selfLength+i+1][0] = input.readFieldValue(FieldMetaInfo{filedType: classType})
	}
	obj.self = t*/

}
func (input *InputObjectStream) readInt(buf *bytes.Buffer) (int) {
	var i int32
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int(i)
}

func (input *InputObjectStream) readInt8(buf *bytes.Buffer) (int8) {
	var i int8
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int8(i)
}

func (input *InputObjectStream) readInt16(buf *bytes.Buffer) (int16) {
	var i int16
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int16(i)
}

func (input *InputObjectStream) readInt32(buf *bytes.Buffer) (int32) {
	var i int32
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int32(i)
}

func (input *InputObjectStream) readFloat32(buf *bytes.Buffer) (float32) {
	var i float32
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return float32(i)
}

func (input *InputObjectStream) readDouble64(buf *bytes.Buffer) (float64) {
	var i float64
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return float64(i)
}

func (input *InputObjectStream) readLong(buf *bytes.Buffer) (int64) {
	var i int64
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int64(i)
}

func (input *InputObjectStream) readInt64(buf *bytes.Buffer) (int64) {
	var i int64
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return int64(i)
}

func (input *InputObjectStream) readUInt16(buf *bytes.Buffer) (uint16) {
	var i uint16
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return uint16(i)
}

func (input *InputObjectStream) readBoolean(buf *bytes.Buffer) (bool) {
	byte, _ := input.in.ReadByte()

	return byte != 0
}

func (input *InputObjectStream) readFieldValue(field FieldMetaInfo) interface{} {

	if field.filedType == "B" {
		return input.readInt8(input.in)
	}

	if field.filedType == "C" {
		return input.readInt16(input.in)
	}

	if field.filedType == "I" {
		return input.readInt32(input.in)
	}

	if field.filedType == "D" {
		return input.readDouble64(input.in)
	}

	if field.filedType == "F" {
		return input.readFloat32(input.in)
	}

	if field.filedType == "J" {
		return input.readInt(input.in)
	}

	if field.filedType == "S" {
		return input.readInt16(input.in)
	}

	if field.filedType == "Z" {
		byte, _ := input.in.ReadByte()

		return byte != 0
	}

	if field.filedType == "L" {
		return input.ReadContent()
	}

	if field.filedType == "[" {
		return input.ReadContent()
	} else {

		return nil
	}

}

//读取非动态代理类的描述信息
func (input *InputObjectStream) readNonProxyDesc() interface{} {
	input.in.ReadByte()

	/*var obj = {
	name: this._readUTFString(),
		serialVersionUID: this.readLong().toString()
	};*/

	var obj = ObjectStreamClass{}
	obj.name = input.readUTFString(false)

	log.DefaultLogger.Infof("start to read readNonProxyDesc,%s", obj.name)

	serialVersionUID := input.readInt64(input.in)
	obj.serialVersionUID = strconv.FormatInt(serialVersionUID, 10)

	//hack一下

	var size int
	if input.refs == nil {
		size = 0
	} else {
		size = len(input.refs)
	}

	input.newHandle(obj)

	var descInfo = input.readClassDescInfo()
	obj.flags = descInfo.flags
	obj.fields = descInfo.fields
	obj.superClass = descInfo.superClass

	input.refs[size] = obj

	log.DefaultLogger.Infof("end to read readNonProxyDesc,%+v", obj)

	return obj
}

//读取类描述,不包括名称的序列化 id
func (input *InputObjectStream) readClassDescInfo() *ObjectStreamClass {

	log.DefaultLogger.Infof("start to read readClassDescInfo")

	var obj = &ObjectStreamClass{}
	obj.flags = input.readClassDescFlags()
	obj.fields = input.readFields()
	input.readClassAnnotation()
	obj.superClass = input.readSuperClassDesc()

	log.DefaultLogger.Infof("end to read readClassDescInfo,%+v", obj.fields)

	return obj
}
func (input *InputObjectStream) readUTFString(isLong bool) string {

	var len int64
	if isLong {
		len = input.readInt64(input.in)
	} else {
		len = int64(input.readUInt16(input.in))
	}

	strBytes, _ := input.ReadBytes(input.in, len)
	return string(strBytes)
}

func (input *InputObjectStream) ReadBytes(buf *bytes.Buffer, size int64) ([]byte, error) {
	tempbytes := make([]byte, size)
	var s, n int = 0, 0
	var err error
	for s < int(size) && err == nil {
		n, err = buf.Read(tempbytes[s:])
		s += n
	}
	return tempbytes, err
}

func (input *InputObjectStream) readClassDescFlags() byte {

	var flag, err = input.in.ReadByte()
	if err != nil {
		return '0'
	} else {
		return flag
	}
}
func (input *InputObjectStream) readFields() []*FieldMetaInfo {
	// fields:
	//   (short)<count>  fieldDesc[count]
	var count = input.readShort(input.in)
	fieldsDesc := make([]*FieldMetaInfo, count) //创建一个大小为xlen+ylen的数组切片

	for i := 0; i < int(count); i++ {
		fieldsDesc[i] = input.readFieldDesc()
	}

	log.DefaultLogger.Infof("read fields,%+v", fieldsDesc)
	return fieldsDesc
}
func (input *InputObjectStream) readShort(buf *bytes.Buffer) (uint16) {
	var i uint16
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0
	}
	return uint16(i)
}
func (input *InputObjectStream) readFieldDesc() *FieldMetaInfo {
	clazzType, _ := input.in.ReadByte()

	var desc = &FieldMetaInfo{}

	desc.filedType = string(clazzType)
	desc.name = input.readUTFString(false)

	if clazzType == '[' || clazzType == 'L' {
		//这里没看懂
		desc.className = input.readObject().(string)
	}

	return desc
}
func (input *InputObjectStream) readClassAnnotation() {
	clazzType, _ := input.in.ReadByte()
	if clazzType == TC_ENDBLOCKDATA {
		log.DefaultLogger.Infof("< _readClassAnnotation | hint = endBlockData")
	} else {
		log.DefaultLogger.Errorf("< _readClassAnnotation | hint = endBlockData")
	}
}

func (input *InputObjectStream) readSuperClassDesc() *ObjectStreamClass {

	log.DefaultLogger.Infof("start to read parent class")

	var superClass = input.readClassDesc()

	log.DefaultLogger.Infof("end to read parent class,%+v", superClass)

	return superClass
}
func (input *InputObjectStream) defaultReadFields(obj *MiddleClass) {

	fields := concatFields(obj.metaInfo)

	if obj.self==nil {
		obj.self = make([]interface{}, 1)
	}

	log.DefaultLogger.Infof("start new reflect type,%+s", obj.name)

	serialize := GetReflectInstance().GetObject(obj.name)
	reflectType := reflect.TypeOf(serialize).Elem()
	log.DefaultLogger.Infof("new reflect type,%+s", reflectType)

	currentObj := reflect.New(reflectType)
	log.DefaultLogger.Infof("new reflect object,%+v", currentObj)

	//这里要条件判断不同的类型

	s := currentObj

	for i := 0; i < len(fields); i++ {
		field := fields[i:i+1][0]
		//读到的值

		var val = input.readFieldValue(*field)

		log.DefaultLogger.Infof("read field type,%+v,value,%+v", field, val)

		sliceValue := reflect.ValueOf(val) // 这里将slice转成reflect.Value类型

		i2 := "Set" + field.name

		log.DefaultLogger.Infof("method=%s", i2)

		method := s.MethodByName(i2)

		v := make([]reflect.Value, 0)

		if field.filedType == "L" && val == nil {
			sliceValue = reflect.ValueOf("")
		}
		v = append(v, sliceValue)

		method.Call(v)

	}

	obj.self = s.Interface()

}
