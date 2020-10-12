package store

import (
	"errors"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"net"
	"time"
)

func SendInheritConfig() error {
	configData, err := Dump()
	if err != nil {
		return err
	}

	var unixConn net.Conn
	// retry 10 time
	for i := 0; i < 10; i++ {
		unixConn, err = net.DialTimeout("unix", types.TransferMosnConfigDomainSocket, 1*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.DefaultLogger.Errorf("[admin store] SendInheritConfig Dial unix failed %v", err)
		return err
	}

	uc := unixConn.(*net.UnixConn)
	defer uc.Close()

	n, err := uc.Write(configData)
	if err != nil {
		log.DefaultLogger.Errorf("[admin store] Write: %v", err)
		return err
	}
	if n != len(configData){
		log.DefaultLogger.Errorf("[server] Write = %d, want %d", n, len(configData))
		return errors.New("write config data length error")
	}

	return nil
}
