package v2

import (
	"fmt"
	"math/big"
	"net"
	//"github.com/alipay/sofa-mosn/pkg/log"
)

func Create(address string, length uint32) CidrRange {
	ipRange := CidrRange{}
	ipRange.create(address, length)
	ipRange.truncateIpAddressAndLength()
	return ipRange
}

func (c *CidrRange) create(address string, length uint32) {
	c.Address = address
	c.Length = length
}

func (c *CidrRange) truncateIpAddressAndLength() {
	ipv4 := net.ParseIP(c.Address).To4()
	if ipv4 != nil {
		if c.Length > 32 {
			c.Length = 32
			return
		}
		if c.Length <= 0 {
			c.Length = 0
			c.Address = "0.0.0.0"
			return
		}
		truncateIp := c.inetAtoN(c.Address) & (^(uint32(0)) << (32 - c.Length))
		c.Address = c.inetNtoA(truncateIp)
	} else {
		//log.DefaultLogger.Errorf("CidrRange truncate fail,currently only support ipv4,address = %v", c.Address)
	}
}

func (c *CidrRange) inetAtoN(ip string) uint32 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return uint32(ret.Uint64())
}
func (c *CidrRange) inetNtoA(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}
func (c *CidrRange) IsInRange(ip string) bool {
	ipv4 := net.ParseIP(c.Address).To4()
	if ipv4 != nil {
		iAddr := c.inetAtoN(ip)
		if iAddr>>(32-c.Length) == (c.inetAtoN(c.Address) >> (32 - c.Length)) {
			return true
		}
		return false
	}
	return false
}
