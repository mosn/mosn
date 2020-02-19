package plugin

import (
	"errors"
	"fmt"
	"net/http"
)

// CheckPluginStatus check plugin's status
func CheckPluginStatus(name string) (string, error) {
	if name == "all" {
		pluginLock.Lock()
		msg := ""
		for name, client := range pluginFactories {
			enable, on := client.Status()
			msg += fmt.Sprintf("name:%s,enable:%t,on:%t\n", name, enable, on)
		}
		pluginLock.Unlock()
		return msg, nil

	} else {
		pluginLock.Lock()
		client := pluginFactories[name]
		pluginLock.Unlock()

		if client == nil {
			return "", errors.New("pulgin " + name + " no register")
		}

		enable, on := client.Status()
		return fmt.Sprintf("name:%s,enable:%t,on:%t", name, enable, on), nil
	}
}

// ClosePlugin disable plugin
func ClosePlugin(name string) error {
	pluginLock.Lock()
	client := pluginFactories[name]
	pluginLock.Unlock()

	if client == nil {
		return errors.New("pulgin " + name + " no register")
	}
	return client.disable()
}

// OpenPlugin open plugin
func OpenPlugin(name string) error {
	pluginLock.Lock()
	client := pluginFactories[name]
	pluginLock.Unlock()

	if client == nil {
		return errors.New("pulgin " + name + " no register")
	}
	return client.open()
}

// http://ip:port/plugin?enable=pluginname
// http://ip:port/plugin?disable=pluginname
// http://ip:port/plugin?status=pluginname
// http://ip:port/plugin?status=all
// NewHttp new http server
func NewHttp(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", AdminApi)
	mux.HandleFunc("/plugin", AdminApi)

	srv := &http.Server{Addr: addr, Handler: mux}
	return srv, nil
}

func AdminApi(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if p := query.Get("enable"); p != "" {
		err := OpenPlugin(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "enable "+p+" success")
		}
	} else if p := query.Get("disable"); p != "" {
		err := ClosePlugin(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "disable "+p+" success")
		}
	} else if p := query.Get("status"); p != "" {
		msg, err := CheckPluginStatus(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, msg)
		}
	} else {
		// help
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Usage:")
		fmt.Fprintln(w, "/plugin?status=all")
		fmt.Fprintln(w, "/plugin?status=pluginname")
		fmt.Fprintln(w, "/plugin?enable=pluginname")
		fmt.Fprintln(w, "/plugin?disable=pluginname")
	}
}
