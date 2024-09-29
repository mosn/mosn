package mosn

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"syscall"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
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

func (op *MosnOperator) Start(params ...string) error {
	spec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	defaultParams := []string{op.binPath, "start", "-c", op.configPath}
	if len(params) > 0 {
		defaultParams = append(defaultParams, params...)
	}
	pid, err := syscall.ForkExec(op.binPath, defaultParams, spec)
	if err != nil {
		return err
	}
	op.pid = pid
	return nil
}

func (op *MosnOperator) Stop() {
	syscall.Kill(op.pid, syscall.SIGKILL) // force kill mosn
}

func (op *MosnOperator) GracefulStop() {
	syscall.Kill(op.pid, syscall.SIGTERM) // graceful stop mosn
}

// TODO: More MOSN Operation
// Restart mosn, hot restart
func (op *MosnOperator) Restart() error {
	return nil
}

func (op *MosnOperator) UpdateConfig(port int, typ string, config string) error {
	// test case, do not care about performance
	body := fmt.Sprintf(`{
		"type": "%s",
		"config": %s
	}`, typ, config)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/debug/update_config", port), strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("request failed")
	}
	return nil
}

func (op *MosnOperator) UpdateRoute(port int, typ string, config string) error {
	// test case, do not care about performance
	body := fmt.Sprintf(`{
		"type": "%s",
		"config": %s
	}`, typ, config)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/debug/update_route", port), strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("request failed")
	}
	return nil
}

func (op *MosnOperator) GetMosnConfig(port int, params string) ([]byte, error) {
	if len(params) > 0 {
		params = "?" + params
	}
	req := fmt.Sprintf("http://127.0.0.1:%d/api/v1/config_dump%s", port, params)
	resp, err := http.Get(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (op *MosnOperator) GetMosnMetrics(port int, key string) ([]byte, error) {
	if len(key) > 0 {
		key = "?key=" + key
	}
	req := fmt.Sprintf("http://127.0.0.1:%d/api/v1/stats%s", port, key)
	resp, err := http.Get(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (op *MosnOperator) LoadMosnConfig() *v2.MOSNConfig {
	return configmanager.Load(op.configPath)
}

var binPath = flag.String("m", "", "-m={mosn_binary_path}")

func StartMosn(cfgStr string, params ...string) *MosnOperator {
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
	if err := mosn.Start(params...); err != nil {
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
