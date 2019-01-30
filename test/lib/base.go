package lib

import (
	"flag"
	"fmt"
	"os"
)

var binPath = flag.String("m", "", "-m={mosn_binary_path}")

func StartMosn(cfgStr string) *MosnOperator {
	flag.Parse()
	if *binPath == "" {
		fmt.Println("no mosn specified")
		os.Exit(1)
	}
	if err := WriteTestConfig(cfgStr); err != nil {
		fmt.Println("write mosn config failed,", err)
		os.Exit(1)
	}
	mosn := NewMosnOperator(*binPath, TempTestConfig)
	if err := mosn.Start(); err != nil {
		fmt.Println("mosn started failed: ", err)
		os.Exit(1)
	}
	return mosn
}

const TempTestConfig = "/tmp/mosn_test_config.json"

func WriteTestConfig(str string) error {
	// create if not exists, or overwrite if exists
	f, err := os.OpenFile(TempTestConfig, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = f.WriteString(str)
	return err
}
