package endpoint

import (
	"fmt"
	"testing"
)

//TestParse tests pasing the endpoint.
func TestParse(t *testing.T) {
	e := Parse("tcp -h 127.0.0.1 -p 19386 -t 60000")
	fmt.Println(e)
	e2 := Parse("udp -h 127.0.0.1 -p 19386 -t 60000")
	fmt.Println(e2)
	tars := Endpoint2tars(e2)
	fmt.Println(tars)
	fmt.Println(Tars2endpoint(tars))
}
