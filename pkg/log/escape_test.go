package log

import (
	"testing"
)

func BenchmarkWrapper(b *testing.B) {
	logger := &logger{
		Output: "stdout",
		Level:  ERROR,
		roller: DefaultRoller(),
	}

	for n := 0; n < b.N; n++ {
		logger.Debugf("say something:%s%s", "hello", expr("world", "!"))
	}

}

func BenchmarkDirect(b *testing.B) {
	logger := &logger{
		Output: "stdout",
		Level:  ERROR,
		roller: DefaultRoller(),
	}

	for n := 0; n < b.N; n++ {
		if logger.Level >= DEBUG {
			logger.Debugf("say something:%s%s", "hello", expr("world", "!"))
		}
	}
}

func expr(left, right string) string {
	return left + right
}
