package featuregate

import "testing"

func TestXdsMtlsEnable(t *testing.T) {
	// Don't parse the flag, assert defaults are used.
	var f = NewFeatureGate()
	f.Add(map[Feature]FeatureSpec{
		XdsMtlsEnable: {Default: true, PreRelease: Beta},
	})

	if f.Enabled(XdsMtlsEnable) != true {
		t.Errorf("Expected true")
	}

	f.Set("XdsMtlsEnable=false")
	if f.Enabled(XdsMtlsEnable) != false {
		t.Errorf("Expected false")
	}

	f.Set("XdsMtlsEnable=true")
	if f.Enabled(XdsMtlsEnable) != true {
		t.Errorf("Expected true")
	}
}
