package lib

import (
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
