package tools

// UniqueInts Return unique element list of list a
func UniqueInts(a []int) (b []int) {
	m := map[int]bool{}
	for _, v := range a {
		if _, ok := m[v]; !ok {
			b = append(b, v)
			m[v] = true
		}
	}
	return b
}
