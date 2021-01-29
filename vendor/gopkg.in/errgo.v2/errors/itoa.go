package errors

// itoa converts val to a decimal string. It allows this package to
// avoid a dependency on strconv. Adapted from net/parse.go
func appendInt(buf []byte, val int) []byte {
	if val < 0 {
		buf = append(buf, '-')
		return appendUint(buf, uint(-val))
	}
	return appendUint(buf, uint(val))
}

func appendUint(p []byte, val uint) []byte {
	var buf [20]byte // big enough for 64bit value base 10
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	// val < 10
	buf[i] = byte('0' + val)
	return append(p, buf[i:]...)
}
