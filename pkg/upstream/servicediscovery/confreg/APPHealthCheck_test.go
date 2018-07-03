package registry

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"testing"
	"time"
)

func init() {
	log.InitDefaultLogger("", log.DEBUG)
}
func TestTestStartAppHealthCheck(t *testing.T) {

	StartAppHealthCheck("http://instpay.rz00b.alipay.net:9500/checkService")
	time.Sleep(3600 * time.Second)
}
