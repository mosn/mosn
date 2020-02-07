package v2

import (
	"fmt"
	"net"
)

// Create CidrRange
func Create(address string, length uint32) *CidrRange {
	// TODO do more check ?
	_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", address, length))
	if err == nil {
		ipRange := &CidrRange{
			Address: address,
			Length:  length,
			IpNet:   ipNet,
		}
		return ipRange
	}
	return nil
}

// IsInRange
func (c *CidrRange) IsInRange(ip net.IP) bool {
	if c.IpNet == nil {
		_, ipNet, err := net.ParseCIDR(fmt.Sprintf("%s/%d", c.Address, c.Length))
		if err == nil {
			c.IpNet = ipNet
		} else {
			return false
		}
	}
	return c.IpNet.Contains(ip)
}
