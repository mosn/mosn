package main

import (
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/champly/lib4go/security/md5"
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
				klog.Infof("receive file success {%s}, md5 is :%s\n\n\n", header.Filename, md5.Encrypt(buf.String()))
				w.Write([]byte("ok"))
			}),
		})
	}
}

// curl -v --http2-prior-knowledge http://localhost:8080 --form 'demo=@/Users/champly/Downloads/idc-test'
