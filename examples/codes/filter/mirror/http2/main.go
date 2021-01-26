package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net"
	"net/http"

	"golang.org/x/net/http2"
	"k8s.io/klog"
)

func main() {
	server := http2.Server{}

	l, err := net.Listen("tcp", ":18080")
	if err != nil {
		klog.Fatal(err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			klog.Fatalf("accept err:%+v", err)
		}

		server.ServeConn(conn, &http2.ServeConnOpts{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				klog.Infof("headers:%+v", r.Header)
				file, header, err := r.FormFile("demo")
				if err != nil {
					klog.Errorf("form file err:%+v", err)
					return
				}
				defer file.Close()

				klog.Infof("receive file:%s", header.Filename)

				var buf bytes.Buffer
				io.Copy(&buf, file)
				klog.Infof("receive file success {%s}, md5 is :%s\n\n\n", header.Filename, Encrypt(buf.String()))
				w.Write([]byte("ok"))
			}),
		})
	}
}

func Encrypt(s string) string {
	return hex.EncodeToString(EncryptBytes([]byte(s)))
}

func EncryptBytes(buffer []byte) []byte {
	m := md5.New()
	m.Write(buffer)
	return m.Sum(nil)
}

// curl -v --http2-prior-knowledge http://localhost:8080 --form 'demo=@/Users/champly/Downloads/idc-test'
