package istio

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetPodLabels(t *testing.T) {
	IstioPodInfoPath = "/tmp/test/pod"
	os.RemoveAll(IstioPodInfoPath)
	err := os.MkdirAll(IstioPodInfoPath, 0755)
	require.Nil(t, err)
	data := []byte("labela=1\nlabelb=2\nlabelc=\"c\"")
	err = ioutil.WriteFile(path.Join(IstioPodInfoPath, "labels"), data, 0644)
	require.Nil(t, err)
	labels := GetPodLabels()
	require.Len(t, labels, 3)
	require.Equal(t, "1", labels["labela"])
	require.Equal(t, "2", labels["labelb"])
	require.Equal(t, "c", labels["labelc"])

}
