package serialize

import (
	"testing"
)

func BenchmarkSerializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	for n := 0; n < b.N; n++ {
		Instance.Serialize(headers)
	}
}

func BenchmarkSerializeString(b *testing.B) {
	className := "com.alipay.sofa.rpc.core.request.SofaRequest"

	for n := 0; n < b.N; n++ {
		Instance.Serialize(className)
	}
}

func BenchmarkDeSerializeMap(b *testing.B) {
	headers := map[string]string{
		"service": "com.alipay.test.TestService:1.0",
	}

	buf, _ := Instance.Serialize(headers)
	header := make(map[string]string)

	for n := 0; n < b.N; n++ {
		Instance.DeSerialize(buf, &header)
	}
}

func BenchmarkDeSerializeString(b *testing.B) {
	className := "com.alipay.sofa.rpc.core.request.SofaRequest"

	buf, _ := Instance.Serialize(className)
	var class string

	for n := 0; n < b.N; n++ {
		Instance.DeSerialize(buf, &class)
	}
}
