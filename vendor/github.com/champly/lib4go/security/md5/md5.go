package md5

import (
	"crypto/md5"
	"encoding/hex"
)

func Encrypt(s string) string {
	return hex.EncodeToString(EncryptBytes([]byte(s)))
}

func EncryptBytes(buffer []byte) []byte {
	m := md5.New()
	m.Write(buffer)
	return m.Sum(nil)
}
