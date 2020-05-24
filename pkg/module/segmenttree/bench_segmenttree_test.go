package segmenttree

import "testing"

func BenchmarkTree_NewTree(b *testing.B) {
	nodeCount := 2000
	nodes := []Node{}
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, Node{
			Value:      i,
			RangeStart: uint64(i),
			RangeEnd:   uint64(i + 1),
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		NewTree(nodes, func(leftChildData, rightChildData interface{}) (currentNodeData interface{}) {
			return leftChildData.(int) + rightChildData.(int)
		})
	}
}

func BenchmarkTree_Update(b *testing.B) {
	nodeCount := 10000
	nodes := []Node{}
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, Node{
			Value:      i,
			RangeStart: uint64(i),
			RangeEnd:   uint64(i + 1),
		})
	}
	t := NewTree(nodes, func(leftChildData, rightChildData interface{}) (currentNodeData interface{}) {
		return leftChildData.(int) + rightChildData.(int)
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l, err := t.Leaf(100)
		if err != nil {
			b.Error(err)
		}
		l.Value = l.Value.(int) + 1
		t.Update(l)
	}

}
