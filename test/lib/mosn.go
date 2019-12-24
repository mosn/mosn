package lib

import (
	"flag"
	"fmt"
	"os"
	"syscall"
)

type MosnOperator struct {
	binPath    string // mosn binary path
	configPath string // mosn config path
	pid        int    // mosn process id
}

func NewMosnOperator(binpath, cfgpath string) *MosnOperator {
	operator := &MosnOperator{
		binPath:    binpath,
		configPath: cfgpath,
	}
	return operator
}

func (op *MosnOperator) Start() error {
	spec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}
	pid, err := syscall.ForkExec(op.binPath, []string{op.binPath, "start", "-c", op.configPath}, spec)
	if err != nil {
		return err
	}
	op.pid = pid
	return nil
}

func (op *MosnOperator) Stop() {
	syscall.Kill(op.pid, syscall.SIGKILL) // force kill mosn
}

// TODO: More MOSN Operation
// Restart mosn, hot restart
func (op *MosnOperator) Restart() error {
	return nil
}

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
