package types

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const podLabelsSeparator = "="

var (
	// IstioPodInfoPath Sidecar mountPath
	IstioPodInfoPath = "/etc/istio/pod"

	labels = make(map[string]string)

	once sync.Once
)

// GetPodLabels return podLabels map
func GetPodLabels() map[string]string {
	once.Do(func() {
		// Add pod labels into nodeInfo
		podLabelsPath := fmt.Sprintf("%s/labels", IstioPodInfoPath)
		if _, err := os.Stat(podLabelsPath); err == nil || os.IsExist(err) {
			f, err := os.Open(podLabelsPath)
			if err == nil {
				defer f.Close()

				br := bufio.NewReader(f)
				for {
					l, _, e := br.ReadLine()
					if e == io.EOF {
						break
					}
					// group="blue"
					keyValueSep := strings.SplitN(strings.ReplaceAll(string(l), "\"", ""), podLabelsSeparator, 2)
					if len(keyValueSep) != 2 {
						continue
					}
					labels[keyValueSep[0]] = keyValueSep[1]
				}
			}
		}
	})

	return labels
}
