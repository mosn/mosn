package serialize

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

//singleton
var Instance = SimpleSerialization{}

type SimpleSerialization struct {
}

func (s *SimpleSerialization) GetSerialNum() int {
	return 6
}

func (s *SimpleSerialization) Serialize(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte{0}, nil
	}
	var rv reflect.Value
	if nrv, ok := v.(reflect.Value); ok {
		rv = nrv
	} else {
		rv = reflect.ValueOf(v)
	}

	t := rv.Type().String()
	//t := fmt.Sprintf("%s", rv.Type())

	buf := new(bytes.Buffer)
	var err error
	switch t {
	case "string":
		_, err = encodeString(rv, buf)
	case "map[string]string":
		//buf.WriteByte(2)
		err = encodeMap(rv, buf)
	case "[]uint8":
		buf.WriteByte(3)
		err = encodeBytes(rv, buf)
	}
	return buf.Bytes(), err
}

func (s *SimpleSerialization) DeSerialize(b []byte, v interface{}) (interface{}, error) {
	if len(b) == 0 {
		return nil, nil
	}
	buf := bytes.NewBuffer(b)
	//tp, _ := buf.ReadByte()

	if sv, ok := v.(*string); ok {
		st, _, err := decodeString(buf)
		if err != nil {
			return nil, err
		}
		if v != nil {
			if sv, ok := v.(*string); ok {
				*sv = st
			}
		}
		return sv, err
	}

	if mv, ok := v.(*map[string]string); ok {
		ma, err := decodeMap(buf)
		if err != nil {
			return nil, err
		}
		if v != nil {
			if mv, ok := v.(*map[string]string); ok {
				*mv = ma
			}
		}
		return mv, err
	}

	return nil, nil

	/*

		case 0:
			v = nil
			return nil, nil
		case 1:
			st, _, err := decodeString(buf)
			if err != nil {
				return nil, err
			}
			if v != nil {
				if sv, ok := v.(*string); ok {
					*sv = st
				}
			}
			return st, err
		case 2:
			ma, err := decodeMap(buf)
			if err != nil {
				return nil, err
			}
			if v != nil {
				if mv, ok := v.(*map[string]string); ok {
					*mv = ma
				}
			}
			return ma, err
		case 3:
			by, err := decodeBytes(buf)
			if err != nil {
				return nil, err
			}
			if v != nil {
				if bv, ok := v.(*[]byte); ok {
					*bv = by
				}
			}
			return by, err
		}

	*/
	//return nil, fmt.Errorf("can not deserialize. unknown type:%v", tp)
}

func (s *SimpleSerialization) SerializeMulti(v []interface{}) ([]byte, error) {
	// TODO support multi value
	if len(v) == 0 {
		return nil, nil
	}
	if len(v) == 1 {
		return s.Serialize(v[0])
	}
	return nil, errors.New("do not support multi value in SimpleSerialization")
}

func (s *SimpleSerialization) DeSerializeMulti(b []byte, v []interface{}) (ret []interface{}, err error) {
	//TODO support multi value
	var rv interface{}
	if v != nil {
		if len(v) == 0 {
			return nil, nil
		}
		if len(v) > 1 {
			return nil, errors.New("do not support multi value in SimpleSerialization")
		}
		rv, err = s.DeSerialize(b, v[0])
	} else {
		rv, err = s.DeSerialize(b, nil)
	}
	return []interface{}{rv}, err
}

func readInt32(buf *bytes.Buffer) (int, error) {
	var i int32
	err := binary.Read(buf, binary.BigEndian, &i)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

func decodeString(buf *bytes.Buffer) (string, int, error) {

	b := buf.Bytes()

	return string(b), buf.Len(), nil
}

/**
这个map是参考com.alipay.sofa.rpc.remoting.codec.SimpleMapSerializer进行实现的.
*/
func decodeMap(buf *bytes.Buffer) (map[string]string, error) {

	result := make(map[string]string, 16)

	for {
		length, err := readInt32(buf)
		if length == -1 || err != nil {
			return result, nil
		}

		key := make([]byte, length)

		buf.Read(key)

		length, err = readInt32(buf)
		if length == -1 || err != nil {
			return result, nil
		}

		value := make([]byte, length)

		buf.Read(value)

		keyStr := string(key)
		valueStr := string(value)

		result[keyStr] = valueStr
	}

}

func decodeBytes(buf *bytes.Buffer) ([]byte, error) {
	size, err := readInt32(buf)
	if err != nil {
		return nil, err
	}
	b := buf.Next(size)
	if len(b) != size {
		return nil, errors.New("read byte not enough")
	}

	return b, nil
}

func encodeStringMap(v reflect.Value, buf *bytes.Buffer) (int32, error) {
	b := []byte(v.String())
	l := int32(len(b))
	err := binary.Write(buf, binary.BigEndian, l)
	err = binary.Write(buf, binary.BigEndian, b)
	if err != nil {
		return 0, err
	}
	return l + 4, nil
}

func encodeString(v reflect.Value, buf *bytes.Buffer) (int32, error) {
	b := []byte(v.String())
	l := int32(len(b))
	//err := binary.Write(buf, binary.BigEndian, l)
	err := binary.Write(buf, binary.BigEndian, b)
	if err != nil {
		return 0, err
	}
	return l + 4, nil
}

func encodeMap(v reflect.Value, buf *bytes.Buffer) error {
	b := new(bytes.Buffer)
	var size, l int32
	var err error
	for _, mk := range v.MapKeys() {
		mv := v.MapIndex(mk)
		l, err = encodeStringMap(mk, b)
		size += l
		if err != nil {
			return err
		}
		l, err = encodeStringMap(mv, b)
		size += l
		if err != nil {
			return err
		}
	}
	err = binary.Write(buf, binary.BigEndian, b.Bytes()[:size])
	return err
}

func encodeBytes(v reflect.Value, buf *bytes.Buffer) error {
	l := len(v.Bytes())
	err := binary.Write(buf, binary.BigEndian, int32(l))
	err = binary.Write(buf, binary.BigEndian, v.Bytes())
	return err
}
