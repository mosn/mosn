/* log_test.go - test for log.go */
package gxlog

import (
	"testing"
)

func TestColorLog(t *testing.T) {
	CDebug("Debug")
	CInfo("Info")
	// CWarn("Warn")
	CWarn("%s", "/test/group%3Dbjtelecom%26protocol%3Dpb%26role%3DSRT_Provider%26service%3Dshopping%26version%3D1.0.1")
	CError("Error")
}
