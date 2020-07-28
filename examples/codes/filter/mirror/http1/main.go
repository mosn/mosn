package main

import (
	"bytes"
	"io"
	"net/http"

	"github.com/champly/lib4go/security/md5"
	"k8s.io/klog"
)

func main() {
	http.HandleFunc("/", handler)

	if err := http.ListenAndServe(":9090", nil); err != nil {
		klog.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
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
}

// curl -v http://localhost:8080 --form 'demo=@/Users/champly/Downloads/idc-test'
