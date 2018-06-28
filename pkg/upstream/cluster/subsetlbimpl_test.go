package cluster

import (
	"math/rand"
	"testing"

	"fmt"
	"github.com/deckarep/golang-set"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

type SortedSet struct {
	Keys mapset.Set
}


func (ss *SortedSet) Len() int {
	return len(ss.Keys.ToSlice())
}

func (ss *SortedSet) Less(i, j int) bool {
	keys := ss.Keys.ToSlice()
	keyi,_:= keys[i].(string)
	keyj, _ := keys[j].(string)

	return keyi < keyj
}

func (ss *SortedSet) Swap(i, j int) {
	keys:= ss.Keys.ToSlice()
	keys[i], keys[j] = keys[j], keys[i]
}

func InnerCompute(i int)int {
	return i + 1
}

type functest func(int)int

func testFunc(f functest,i int) int {
	return f(i)
}

func TestLoadBalancerSubsetInfoImpl_SubsetKeys(t *testing.T) {
	
	s := func(i int) int {
		return InnerCompute(i)
	}

	fmt.Println(testFunc(s,2))
	
	////s := []string{"test","testa","china"}
	//requiredClasses := mapset.NewSet()
	//requiredClasses.Add("Cooking")
	//requiredClasses.Add("English")
	//requiredClasses.Add("Math")
	//requiredClasses.Add("Biology")
	//
	//ss := SortedSet{
	//	Keys:requiredClasses,
	//}
	//
	//sort.Sort(&ss)
	//
	//fmt.Println(requiredClasses.ToSlice())
	////fmt.Println(requiredClasses.ToSlice()[0])
	////optionalClasses := mapset.NewSet()
	////optionalClasses.Add("Math")
	////optionalClasses.Add("Biology")
	//
	////var subSetKeys = []mapset.Set{requiredClasses, optionalClasses}
	//
	//fmt.Println(ss)
	
	
	
}

func Test_subSetLoadBalancer_FindOrCreateSubset(t *testing.T) {

	var m = map[string]string{"test":"1"}

	
	fmt.Println(m)
	
	type fields struct {
		lbType              types.LoadBalancerType
		subSetKeys          [][]string
		fallbackSubset      *LBSubsetEntry
		runtime             types.Loader
		stats               *types.ClusterStats
		random              rand.Rand
		fallBackPolicy      types.FallBackPolicy
		defaultSubSet       []types.Host
		originalPrioritySet types.PrioritySet
		subSets             types.LbSubsetMap
	}
	type args struct {
		subsets types.LbSubsetMap
		kvs     types.SubsetMetadata
		idx     uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
