package metrics

type metricsMatcher struct {
	rejectAll       bool
	exclusionLabels []string
	exclusionKeys   []string
}

// isExclusionLabels returns the labels will be ignored or not
func (m *metricsMatcher) isExclusionLabels(labels map[string]string) bool {
	if m.rejectAll {
		return true
	}
	// TODO: support pattern
	for _, label := range m.exclusionLabels {
		if _, ok := labels[label]; ok {
			return true
		}
	}
	return false
}

// isExclusionKey returns the key will be ignored or not
func (m *metricsMatcher) isExclusionKey(key string) bool {
	if m.rejectAll {
		return true
	}
	// TODO: support pattern
	for _, eKey := range m.exclusionKeys {
		if eKey == key {
			return true
		}
	}
	return false
}
