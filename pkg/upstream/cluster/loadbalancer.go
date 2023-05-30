/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"math"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trainyao/go-maglev"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// NewLoadBalancer can be register self defined type
var lbFactories map[types.LoadBalancerType]func(types.ClusterInfo, types.HostSet) types.LoadBalancer

func RegisterLBType(lbType types.LoadBalancerType, f func(types.ClusterInfo, types.HostSet) types.LoadBalancer) {
	if lbFactories == nil {
		lbFactories = make(map[types.LoadBalancerType]func(types.ClusterInfo, types.HostSet) types.LoadBalancer)
	}
	lbFactories[lbType] = f
}

type SlowStartFactorFunc func(info types.ClusterInfo, host types.Host) float64

var slowStartFuncFactories map[types.SlowStartMode]SlowStartFactorFunc

// RegisterSlowStartMode can register self defined modes
func RegisterSlowStartMode(mode types.SlowStartMode, factorFunc SlowStartFactorFunc) {
	if slowStartFuncFactories == nil {
		slowStartFuncFactories = make(map[types.SlowStartMode]SlowStartFactorFunc)
	}
	slowStartFuncFactories[mode] = factorFunc
}

var rrFactory *roundRobinLoadBalancerFactory

func init() {
	rrFactory = &roundRobinLoadBalancerFactory{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	RegisterLBType(types.RoundRobin, rrFactory.newRoundRobinLoadBalancer)
	RegisterLBType(types.Random, newRandomLoadBalancer)
	RegisterLBType(types.WeightedRoundRobin, newWRRLoadBalancer)
	RegisterLBType(types.LeastActiveRequest, newLeastActiveRequestLoadBalancer)
	RegisterLBType(types.Maglev, newMaglevLoadBalancer)
	RegisterLBType(types.RequestRoundRobin, newReqRoundRobinLoadBalancer)
	RegisterLBType(types.LeastActiveConnection, newLeastActiveConnectionLoadBalancer)
	RegisterLBType(types.PeakEwma, newPeakEwmaLoadBalancer)

	RegisterSlowStartMode(types.ModeDuration, slowStartDurationFactorFunc)

	registerVariables()
}

var (
	VarProxyUpstreamIndex = "upstream_index"
)

var (
	buildinVariables = []variable.Variable{
		variable.NewStringVariable(VarProxyUpstreamIndex, nil, nil, variable.DefaultStringSetter, 0),
	}
)

func registerVariables() {
	for idx := range buildinVariables {
		variable.Register(buildinVariables[idx])
	}
}

func NewLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lbType := info.LbType()
	if f, ok := lbFactories[lbType]; ok {
		return f(info, hosts)
	}
	return rrFactory.newRoundRobinLoadBalancer(info, hosts)
}

// LoadBalancer Implementations

type randomLoadBalancer struct {
	mutex sync.Mutex
	rand  *rand.Rand
	hosts types.HostSet
	rrLB  types.LoadBalancer // if node fails, we'll degrade to rr load balancer
}

func newRandomLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	return &randomLoadBalancer{
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		hosts: hosts,
		rrLB:  rrFactory.newRoundRobinLoadBalancer(info, hosts),
	}
}

func (lb *randomLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	hs := lb.hosts
	total := hs.Size()
	if total == 0 {
		return nil
	}

	lb.mutex.Lock()
	idx := lb.rand.Intn(total)
	lb.mutex.Unlock()

	host := hs.Get(idx)
	if host.Health() {
		return host
	}

	// degrade to rr lb, to make node selection more balanced
	return lb.rrLB.ChooseHost(context)
}

func (lb *randomLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.hosts.Size() > 0
}

func (lb *randomLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

type roundRobinLoadBalancer struct {
	hosts   types.HostSet
	rrIndex uint32
}

type roundRobinLoadBalancerFactory struct {
	mutex sync.Mutex
	rand  *rand.Rand
}

func (f *roundRobinLoadBalancerFactory) newRoundRobinLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	var idx uint32
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if hosts.Size() != 0 {
		idx = f.rand.Uint32() % uint32(hosts.Size())
	}
	return &roundRobinLoadBalancer{
		hosts:   hosts,
		rrIndex: idx,
	}
}

func (lb *roundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	hs := lb.hosts
	total := hs.Size()
	if total == 0 {
		return nil
	}
	for i := 0; i < total; i++ {
		index := atomic.AddUint32(&lb.rrIndex, 1) % uint32(total)
		host := hs.Get(int(index))
		if host.Health() {
			return host
		}
	}

	// Reference https://github.com/mosn/mosn/issues/1663
	secondStartIndex := int(atomic.AddUint32(&lb.rrIndex, 1) % uint32(total))
	for i := 0; i < total; i++ {
		index := (i + secondStartIndex) % total
		host := hs.Get(index)
		if host.Health() {
			return host
		}
	}

	return nil
}

func (lb *roundRobinLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.hosts.Size() > 0
}

func (lb *roundRobinLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

/*
A round robin load balancer. When in weighted mode, EDF scheduling is used. When in not
weighted mode, simple RR index selection is used.
*/
type WRRLoadBalancer struct {
	*EdfLoadBalancer
	rrLB types.LoadBalancer
}

func newWRRLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	wrrLB := &WRRLoadBalancer{}
	wrrLB.EdfLoadBalancer = newEdfLoadBalancer(info, hosts, wrrLB.unweightChooseHost, wrrLB.hostWeight)
	wrrLB.rrLB = rrFactory.newRoundRobinLoadBalancer(info, hosts)
	return wrrLB
}

func (lb *WRRLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.hosts.Size() > 0
}

func (lb *WRRLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

func (lb *WRRLoadBalancer) hostWeight(item WeightItem) float64 {
	host := item.(types.Host)
	return fixHostWeight(float64(host.Weight()))
}

// do unweighted (fast) selection
func (lb *WRRLoadBalancer) unweightChooseHost(context types.LoadBalancerContext) types.Host {
	return lb.rrLB.ChooseHost(context)
}

const defaultChoice = 2
const defaultActiveRequestBias = 1.0

// leastActiveRequestLoadBalancer choose the host with the least active request
type leastActiveRequestLoadBalancer struct {
	*EdfLoadBalancer
	choice            uint32
	activeRequestBias float64
}

func newLeastActiveRequestLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lb := &leastActiveRequestLoadBalancer{}
	if info != nil && info.LbConfig() != nil {
		lb.choice = info.LbConfig().ChoiceCount
		lb.activeRequestBias = info.LbConfig().ActiveRequestBias
	} else {
		lb.choice = defaultChoice
		lb.activeRequestBias = defaultActiveRequestBias
	}
	lb.EdfLoadBalancer = newEdfLoadBalancer(info, hosts, lb.unweightChooseHost, lb.hostWeight)
	return lb
}

func (lb *leastActiveRequestLoadBalancer) hostWeight(item WeightItem) float64 {
	host, ok := item.(types.Host)
	if !ok {
		return float64(item.Weight())
	}

	weight := fixHostWeight(float64(host.Weight()))

	biasedActiveRequest := math.Pow(float64(host.HostStats().UpstreamRequestActive.Count())+1, lb.activeRequestBias)

	return weight / biasedActiveRequest
}

func (lb *leastActiveRequestLoadBalancer) unweightChooseHost(context types.LoadBalancerContext) types.Host {

	hs := lb.hosts
	total := hs.Size()
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	var candidate types.Host
	// Choose `choice` times and return the best one
	// See The Power of Two Random Choices: A Survey of Techniques and Results
	//  http://www.eecs.harvard.edu/~michaelm/postscripts/handbook2001.pdf
	for cur := 0; cur < int(lb.choice); cur++ {

		randIdx := lb.rand.Intn(total)
		tempHost := hs.Get(randIdx)
		if candidate == nil {
			candidate = tempHost
			continue
		}
		if candidate.HostStats().UpstreamRequestActive.Count() > tempHost.HostStats().UpstreamRequestActive.Count() {
			candidate = tempHost
		}
	}
	return candidate

}

type EdfLoadBalancer struct {
	scheduler *edfScheduler
	hosts     types.HostSet
	rand      *rand.Rand
	mutex     sync.Mutex
	// the method to choose host when all host
	unweightChooseHostFunc func(types.LoadBalancerContext) types.Host
	hostWeightFunc         func(item WeightItem) float64
}

func (lb *EdfLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {

	var candidate types.Host
	hs := lb.hosts
	total := hs.Size()
	if total == 0 {
		// Return nil directly if allHosts is nil or size is 0
		return nil
	}
	if total == 1 {
		targetHost := hs.Get(0)
		// Return directly if there is only one host
		if targetHost.Health() {
			return targetHost
		}
		return nil
	}

	if lb.scheduler != nil {
		for i := 0; i < total; i++ {
			// do weight selection
			candidate = lb.scheduler.NextAndPush(lb.hostWeightFunc).(types.Host)
			if candidate != nil && candidate.Health() {
				return candidate
			}
		}
	}

	// Use unweighted round-robin as a fallback while failed to pick a healthy host by weighted round-robin.
	return lb.unweightChooseHostFunc(context)
}

func (lb *EdfLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.hosts.Size() > 0
}

func (lb *EdfLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

func newEdfLoadBalancer(info types.ClusterInfo, hosts types.HostSet, unWeightChoose func(types.LoadBalancerContext) types.Host, hostWeightFunc func(host WeightItem) float64) *EdfLoadBalancer {
	hostWeightFunc = slowStartHostWeightFunc(info, hostWeightFunc)
	lb := &EdfLoadBalancer{
		hosts:                  hosts,
		rand:                   rand.New(rand.NewSource(time.Now().UnixNano())),
		unweightChooseHostFunc: unWeightChoose,
		hostWeightFunc:         hostWeightFunc,
	}
	lb.refresh(info, hosts)
	return lb
}

func slowStartDurationFactorFunc(info types.ClusterInfo, host types.Host) float64 {
	return slowStartDurationFactorFuncWithNowFunc(info, host, time.Now)
}

// slowStartDurationFactorFuncWithNowFunc with nowFunc parameter for testing
func slowStartDurationFactorFuncWithNowFunc(info types.ClusterInfo, host types.Host, nowFunc func() time.Time) float64 {
	slowStart := info.SlowStart()

	if slowStart.SlowStartDuration <= 0 {
		return 1.0
	}

	if host.LastHealthCheckPassTime().IsZero() {
		return 1.0
	}

	duration := nowFunc().Sub(host.LastHealthCheckPassTime())
	window := slowStart.SlowStartDuration
	if duration >= window {
		return 1.0
	}

	return math.Max(1.0, duration.Seconds()) / window.Seconds()
}

// slowStartHostWeightFunc progressively increases amount of traffic for newly added upstream hosts
func slowStartHostWeightFunc(info types.ClusterInfo, hostWeightFunc func(host WeightItem) float64) func(host WeightItem) float64 {
	if info == nil {
		return hostWeightFunc
	}

	slowStart := info.SlowStart()

	mode := slowStart.Mode
	if mode == "" {
		return hostWeightFunc
	}

	factorFunc := slowStartFuncFactories[mode]
	if factorFunc == nil {
		log.DefaultLogger.Warnf("[lb][slow_start] Unregistered slow start mode: %s, slow start will not be performed",
			mode)
		return hostWeightFunc
	}

	return func(host WeightItem) float64 {
		w := hostWeightFunc(host)
		h, ok := host.(types.Host)
		if !ok {
			return w
		}

		a := slowStart.Aggression

		f := factorFunc(info, h)
		if f >= 1.0 {
			return w
		}

		if a != 1.0 {
			f = math.Pow(f, 1/a)
		}

		if f < slowStart.MinWeightPercent {
			f = slowStart.MinWeightPercent
		}

		return w * f
	}
}

func (lb *EdfLoadBalancer) refresh(info types.ClusterInfo, hosts types.HostSet) {
	var slowStart types.SlowStart
	if info != nil {
		slowStart = info.SlowStart()
	}

	if hosts.Size() <= 1 {
		return
	}

	// Check if the slow-start not configured and original host weights are equal and skip EDF creation if they are
	if slowStart.Mode == "" && hostWeightsAreEqual(hosts) {
		return
	}

	lb.scheduler = newEdfScheduler(hosts.Size())

	// Init Edf scheduler with healthy hosts.
	hosts.Range(func(host types.Host) bool {
		lb.scheduler.Add(host, lb.hostWeightFunc(host))
		return true
	})
	// refer blog http://zablog.me/2019/08/02/2019-08-02/
	// avoid instance flood pressure for the first entry start from a random one via pick random times
	randomPick := lb.rand.Intn(hosts.Size())
	for i := 0; i < randomPick; i++ {
		lb.scheduler.NextAndPush(lb.hostWeightFunc)
	}
}

func hostWeightsAreEqual(hosts types.HostSet) bool {
	weight := hosts.Get(0).Weight()

	for i := 1; i < hosts.Size(); i++ {
		if hosts.Get(i).Weight() != weight {
			return false
		}
	}
	return true
}

// newMaglevLoadBalancer return maglevLoadBalancer structure.
//
// In maglevLoadBalancer, there is a maglev table for consistence hash host choosing.
// If the chosen host is unhealthy, maglevLoadBalancer will traverse host list to find a healthy host.
func newMaglevLoadBalancer(info types.ClusterInfo, set types.HostSet) types.LoadBalancer {
	names := make([]string, 0, set.Size())
	set.Range(func(host types.Host) bool {
		names = append(names, host.AddressString())
		return true
	})
	mgv := &maglevLoadBalancer{
		hosts: set,
	}

	nameCount := len(names)
	// if host count > BigM, maglev table building will cross array boundary
	// maglev lb will not work in this scenario
	if nameCount >= maglev.BigM {
		log.DefaultLogger.Errorf("[lb][maglev] host count too large, expect <= %d, get %d",
			maglev.BigM, nameCount)
		return mgv
	}
	if nameCount == 0 {
		return mgv
	}

	maglevM := maglev.SmallM
	// according to test, 30000 host with testing 1e8 times, hash distribution begins to go wrong,
	// max=4855, mean=3333.3333333333335, peak-to-mean=1.4565
	// so use BigM when host >= 30000
	limit := 30000
	if nameCount >= limit {
		log.DefaultLogger.Infof("[lb][maglev] host count %d >= %d, using maglev.BigM", nameCount, limit)
		maglevM = maglev.BigM
	}

	mgv.maglev = maglev.New(names, uint64(maglevM))
	return mgv
}

type maglevLoadBalancer struct {
	hosts  types.HostSet
	maglev *maglev.Table
}

func (lb *maglevLoadBalancer) ChooseHost(ctx types.LoadBalancerContext) types.Host {
	// host empty, maglev info may be nil
	if lb.maglev == nil {
		return nil
	}

	route := ctx.DownstreamRoute()
	if route == nil || route.RouteRule() == nil {
		return nil
	}

	hashPolicy := route.RouteRule().Policy().HashPolicy()
	if hashPolicy == nil {
		return nil
	}

	hash := hashPolicy.GenerateHash(ctx.DownstreamContext())
	index := lb.maglev.Lookup(hash)
	chosen := lb.hosts.Get(index)

	// if retry, means request to last chose host failed, do not use it again
	retrying := false
	context := ctx.DownstreamContext()
	if ind, err := variable.GetString(context, VarProxyUpstreamIndex); err == nil {
		if i, err := strconv.Atoi(ind); err == nil {
			index = i
		}
		retrying = true
	}
	// fallback
	if !chosen.Health() || retrying {
		chosen, index = lb.chooseHostFromHostList(index + 1)
	}

	if chosen == nil {
		if log.Proxy.GetLogLevel() >= log.INFO {
			log.Proxy.Infof(ctx.DownstreamContext(), "[lb][maglev] hash %d get nil host, index: %d",
				hash, index)
		}
	} else {
		variable.SetString(context, VarProxyUpstreamIndex, strconv.Itoa(index))
		if log.Proxy.GetLogLevel() >= log.DEBUG {
			log.Proxy.Debugf(ctx.DownstreamContext(), "[lb][maglev] hash %d index %d get host %s",
				hash, index, chosen.AddressString())
		}
	}

	return chosen
}

func (lb *maglevLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.HostNum(metadata) > 0
}

func (lb *maglevLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

// chooseHostFromHostList traverse host list to find a healthy host
func (lb *maglevLoadBalancer) chooseHostFromHostList(index int) (types.Host, int) {
	total := lb.hosts.Size()

	for i := 0; i < total; i++ {
		ind := (index + i) % total
		host := lb.hosts.Get(ind)
		if host.Health() {
			return host, ind
		}
	}

	return nil, index
}

type reqRoundRobinLoadBalancer struct {
	hosts types.HostSet
}

func newReqRoundRobinLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	return &reqRoundRobinLoadBalancer{
		hosts: hosts,
	}
}

// request round robin load balancer choose host start from index 0 every single context, and round robin when reentry
func (lb *reqRoundRobinLoadBalancer) ChooseHost(context types.LoadBalancerContext) types.Host {
	hs := lb.hosts
	total := hs.Size()
	if total == 0 {
		return nil
	}
	ctx := context.DownstreamContext()
	ind := 0
	if index, err := variable.GetString(ctx, VarProxyUpstreamIndex); err == nil {
		if i, err := strconv.Atoi(index); err == nil {
			ind = i + 1
		}
	}
	for id := ind; id < total+ind; id++ {
		idx := id % total
		target := hs.Get(idx)
		if target.Health() {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf("[lb] [RequestRoundRobin] choose host: %s", target.AddressString())
			}
			variable.SetString(ctx, VarProxyUpstreamIndex, strconv.Itoa(idx))
			return target
		}
	}

	return nil
}

func (lb *reqRoundRobinLoadBalancer) IsExistsHosts(metadata api.MetadataMatchCriteria) bool {
	return lb.hosts.Size() > 0
}

func (lb *reqRoundRobinLoadBalancer) HostNum(metadata api.MetadataMatchCriteria) int {
	return lb.hosts.Size()
}

const defaultPeakEwmaDuration = 10 * time.Second

type peakEwmaLoadBalancer struct {
	*EdfLoadBalancer
	rrLB types.LoadBalancer

	choice            uint32
	activeRequestBias float64

	defaultDuration time.Duration
}

func newPeakEwmaLoadBalancer(info types.ClusterInfo, hosts types.HostSet) types.LoadBalancer {
	lb := &peakEwmaLoadBalancer{}
	lb.rrLB = rrFactory.newRoundRobinLoadBalancer(info, hosts)
	lb.EdfLoadBalancer = newEdfLoadBalancer(info, hosts, lb.unweightedChoose, lb.hostWeight)

	if info != nil && info.LbConfig() != nil {
		lb.choice = info.LbConfig().ChoiceCount
		lb.activeRequestBias = info.LbConfig().ActiveRequestBias
	} else {
		lb.choice = defaultChoice
		lb.activeRequestBias = defaultActiveRequestBias
	}

	if info != nil {
		lb.defaultDuration = info.ConnectTimeout() + info.IdleTimeout()
	}

	if lb.defaultDuration == 0 {
		lb.defaultDuration = defaultPeakEwmaDuration
	}

	return lb
}

func (lb *peakEwmaLoadBalancer) hostWeight(item WeightItem) float64 {
	host, ok := item.(types.Host)
	if !ok {
		return fixHostWeight(float64(item.Weight()))
	}

	return fixHostWeight(float64(host.Weight())) / lb.unweightedPeakEwmaScore(host)
}

func (lb *peakEwmaLoadBalancer) unweightedChoose(context types.LoadBalancerContext) types.Host {
	hs := lb.hosts

	var candidate types.Host

	// If `total` is less than or equal to `choice`, we can iterate over all elements directly.
	if hs.Size() <= int(lb.choice) {
		candidate = lb.iterateChoose()
	} else {
		candidate = lb.randomChoose()
		if candidate == nil {
			if log.DefaultLogger.GetLogLevel() >= log.WARN {
				log.DefaultLogger.Warnf("[lb][PeakEwma] no host chosen after %d choice, fallback to RR", lb.choice)
			}

			return lb.rrLB.ChooseHost(context)
		}
	}

	return candidate
}

func (lb *peakEwmaLoadBalancer) iterateChoose() types.Host {
	total := lb.hosts.Size()

	var candidate types.Host

	// There is a special case here, that is, metrics can be disabled,
	// see `pkg/metrics/matcher#metricsMatcher`. If the metrics used to
	// calculate the score of PeakEWMA are disabled, all hosts will get
	// the same score, and will always peek the first host.
	//
	// By choosing from a random index, when metrics are disabled,
	// it can be automatically fallback to random without causing skew.
	lb.mutex.Lock()
	idx := lb.rand.Intn(total)
	lb.mutex.Unlock()

	for i := 0; i < total; i++ {
		temp := lb.hosts.Get((i + idx) % total)
		if !temp.Health() {
			continue
		}

		if candidate == nil || lb.unweightedPeakEwmaScore(temp) < lb.unweightedPeakEwmaScore(candidate) {
			candidate = temp
		}
	}

	if candidate != nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[lb][PeakEwma] iterate choose host %s with score %f",
				candidate.AddressString(), lb.unweightedPeakEwmaScore(candidate))
		}
	}

	return candidate
}

// Choose `choice` times and return the best one.
// See The Power of Two Random Choices: A Survey of Techniques and Results
//
//	http://www.eecs.harvard.edu/~michaelm/postscripts/handbook2001.pdf
func (lb *peakEwmaLoadBalancer) randomChoose() types.Host {
	total := lb.hosts.Size()

	var candidate types.Host

	for i := 0; i < int(lb.choice); i++ {
		// don't ensure uniqueness because it has high cost
		lb.mutex.Lock()
		idx := lb.rand.Intn(total)
		lb.mutex.Unlock()

		temp := lb.hosts.Get(idx)
		if !temp.Health() {
			continue
		}

		if candidate == nil || lb.unweightedPeakEwmaScore(temp) < lb.unweightedPeakEwmaScore(candidate) {
			candidate = temp
		}
	}

	if candidate != nil {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("[lb][PeakEwma] random choose host %s with score %f",
				candidate.AddressString(), lb.unweightedPeakEwmaScore(candidate))
		}
	}

	return candidate
}

const minPeakEwmaSuccessRate = 0.1

func (lb *peakEwmaLoadBalancer) unweightedPeakEwmaScore(h types.Host) float64 {
	stats := h.HostStats()

	duration := stats.UpstreamRequestDurationEWMA.Rate()

	// None of the active requests returned, try to use cluster duration as default.
	if duration == 0 {
		duration = h.ClusterInfo().Stats().UpstreamRequestDurationEWMA.Rate()
		// None of the active requests returned in cluster, or the metrics is disabled
		if duration == 0 {
			duration = float64(lb.defaultDuration)
		}
	}

	biasedActiveRequest := math.Pow(float64(stats.UpstreamRequestActive.Count())+1, lb.activeRequestBias)

	responseTotal := stats.UpstreamResponseTotalEWMA.Rate()
	responseClientError := stats.UpstreamResponseClientErrorEWMA.Rate()
	responseServerError := stats.UpstreamResponseServerErrorEWMA.Rate()

	// add 1 to make sure the error rate is always decaying
	clientErrorRate := responseClientError / (responseTotal + 1)
	serverErrorRate := responseServerError / (responseTotal + 1)

	successRate := (1 - clientErrorRate) * (1 - serverErrorRate)
	if successRate < minPeakEwmaSuccessRate {
		successRate = minPeakEwmaSuccessRate
	}

	return duration * biasedActiveRequest / successRate
}
