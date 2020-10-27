package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// PacketInfo packet info
type PacketInfo struct {
	msg   []byte
	start int
	end   int
}

const (
	udpPacketMaxSize = 64 * 1024
)

var bufPool = sync.Pool{New: func() interface{} { return make([]byte, udpPacketMaxSize) }}

// NewPacketInfo new packet info
func NewPacketInfo() *PacketInfo {
	return &PacketInfo{
		msg:   bufPool.Get().([]byte),
		start: 0,
		end:   0,
	}
}

// Clear clear the packet
func (packet *PacketInfo) Clear() {
	packet.start = 0
	packet.end = 0
}

// Destroy destroy packet
func (packet *PacketInfo) Destroy() {
	bufPool.Put(packet.msg)
}

// SetReusePort set reuseport sockopt
func SetReusePort(fd int) error {
	err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		msg := fmt.Sprintf("set reuseport %d failed: %v", fd, err)
		return errors.New(msg)
	}

	return nil
}

func main() {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				SetReusePort(int(fd))
			})
		},
	}
	nc, err := lc.ListenPacket(context.Background(), "udp", "127.0.0.1:5300")
	if err != nil {
		fmt.Printf("Failed to listen, %s", err)
		return
	}

	fmt.Println("Listening on udp port 5300 ...")

	conn := nc.(*net.UDPConn)
	var packet = NewPacketInfo()
	defer packet.Destroy()

	for {
		packet.Clear()
		n, raddr, err := conn.ReadFromUDP(packet.msg)
		if err != nil {
			fmt.Println("Could not receive packets, err:", err)
			continue
		}
		packet.end = n
		fmt.Printf("Receive from %s, len:%d, data:%s\n", raddr.String(), n, packet.msg[:n])
		n, err = conn.WriteToUDP(packet.msg[packet.start:packet.end], raddr)
		if err != nil {
			fmt.Println("Write to upstream err:", err)
		}
	}
}
