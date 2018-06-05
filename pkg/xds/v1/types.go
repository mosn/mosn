package v1

import (
	"net/http"
	"encoding/json"
	"strings"
	"sort"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

const (
	// DefaultAccessLog is the name of the log channel (stdout in docker environment)
	DefaultAccessLog = "/dev/stdout"

	// DefaultLbType defines the default load balancer policy
	DefaultLbType = LbTypeRoundRobin

	// LDSName is the name of listener-discovery-service (LDS) cluster
	LDSName = "lds"

	// RDSName is the name of route-discovery-service (RDS) cluster
	RDSName = "rds"

	// SDSName is the name of service-discovery-service (SDS) cluster
	SDSName = "sds"

	// CDSName is the name of cluster-discovery-service (CDS) cluster
	CDSName = "cds"

	// RDSAll is the special name for HTTP PROXY route
	RDSAll = "http_proxy"

	// VirtualListenerName is the name for traffic capture listener
	VirtualListenerName = "virtual"

	// ClusterTypeStrictDNS name for clusters of type 'strict_dns'
	ClusterTypeStrictDNS = "strict_dns"

	// ClusterTypeStatic name for clusters of type 'static'
	ClusterTypeStatic = "static"

	// ClusterTypeOriginalDST name for clusters of type 'original_dst'
	ClusterTypeOriginalDST = "original_dst"

	// LbTypeRoundRobin is the name for roundrobin LB
	LbTypeRoundRobin = "round_robin"

	// LbTypeLeastRequest is the name for least request LB
	LbTypeLeastRequest = "least_request"

	// LbTypeRandom is the name for random LB
	LbTypeRandom = "random"

	// LbTypeOriginalDST is the name for LB of original_dst
	LbTypeOriginalDST = "original_dst_lb"

	// ClusterFeatureHTTP2 is the feature to use HTTP/2 for a cluster
	ClusterFeatureHTTP2 = "http2"

	// HTTPConnectionManager is the name of HTTP filter.
	HTTPConnectionManager = "http_connection_manager"

	// TCPProxyFilter is the name of the TCP Proxy network filter.
	TCPProxyFilter = "tcp_proxy"

	// MONGOProxyFilter is the name of the MONGO Proxy network filter.
	MONGOProxyFilter = "mongo_proxy"

	// WildcardAddress binds to all IP addresses
	WildcardAddress = "0.0.0.0"

	// LocalhostAddress for local binding
	LocalhostAddress = "127.0.0.1"

	// IngressTraceOperation denotes the name of trace operation for Envoy
	IngressTraceOperation = "ingress"

	// ZipkinTraceDriverType denotes the Zipkin HTTP trace driver
	ZipkinTraceDriverType = "zipkin"

	// ZipkinCollectorCluster denotes the cluster where zipkin server is running
	ZipkinCollectorCluster = "zipkin"

	// ZipkinCollectorEndpoint denotes the REST endpoint where Envoy posts Zipkin spans
	ZipkinCollectorEndpoint = "/api/v1/spans"

	router  = "router"
	auto    = "auto"
	decoder = "decoder"
	read    = "read"
	both    = "both"
)

type xdsClient struct {
	httpClient *http.Client
	serviceCluster string
	serviceNode string
}

// Tracing definition
type Tracing struct {
	HTTPTracer HTTPTracer `json:"http"`
}

// HTTPTracer definition
type HTTPTracer struct {
	HTTPTraceDriver HTTPTraceDriver `json:"driver"`
}

// HTTPTraceDriver definition
type HTTPTraceDriver struct {
	HTTPTraceDriverType   string                `json:"type"`
	HTTPTraceDriverConfig HTTPTraceDriverConfig `json:"config"`
}

// HTTPTraceDriverConfig definition
type HTTPTraceDriverConfig struct {
	CollectorCluster  string `json:"collector_cluster"`
	CollectorEndpoint string `json:"collector_endpoint"`
}

// RootRuntime definition.
// See https://envoyproxy.github.io/envoy/configuration/overview/overview.html
type RootRuntime struct {
	SymlinkRoot          string `json:"symlink_root"`
	Subdirectory         string `json:"subdirectory"`
	OverrideSubdirectory string `json:"override_subdirectory,omitempty"`
}

// AbortFilter definition
type AbortFilter struct {
	Percent    int `json:"abort_percent,omitempty"`
	HTTPStatus int `json:"http_status,omitempty"`
}

// DelayFilter definition
type DelayFilter struct {
	Type     string `json:"type,omitempty"`
	Percent  int    `json:"fixed_delay_percent,omitempty"`
	Duration int64  `json:"fixed_duration_ms,omitempty"`
}

type RangeMatch struct {
	start int `json:"start, omitempty"`
	end int `json:"end, omitempty"`
}

// Header definition
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Regex bool   `json:"regex,omitempty"`
	RangeMatch *RangeMatch  `json:"range_match,omitempty"`
}

// FilterFaultConfig definition
type FilterFaultConfig struct {
	Abort           *AbortFilter `json:"abort,omitempty"`
	Delay           *DelayFilter `json:"delay,omitempty"`
	Headers         Headers      `json:"headers,omitempty"`
	UpstreamCluster string       `json:"upstream_cluster,omitempty"`
}

// FilterRouterConfig definition
type FilterRouterConfig struct {
	// DynamicStats defaults to true
	DynamicStats bool `json:"dynamic_stats,omitempty"`
}

// HTTPFilter definition
type HTTPFilter struct {
	Type string `json:"type"`
	Name string `json:"name"`
	// Config            interface{} `json:"config"`
	Config            json.RawMessage `json:"config"`
	FilterMixerConfig *FilterMixerConfig
	FilterFaultConfig *FilterFaultConfig
}

// Runtime definition
type Runtime struct {
	Key     string `json:"key"`
	Default int    `json:"default"`
}

// HTTPRoute definition
type HTTPRoute struct {
	Runtime *Runtime `json:"runtime,omitempty"`

	Path   string `json:"path,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Regex  string `json:"regex,omitempty"`

	CaseSensitive bool `json:"case_sensitive"`

	PrefixRewrite string `json:"prefix_rewrite,omitempty"`
	HostRewrite   string `json:"host_rewrite,omitempty"`

	PathRedirect string `json:"path_redirect,omitempty"`
	HostRedirect string `json:"host_redirect,omitempty"`

	Cluster          string           `json:"cluster,omitempty"`
	ClusterHeader    string           `json:"cluster_header,omitempty"`
	WeightedClusters *WeightedCluster `json:"weighted_clusters,omitempty"`

	Headers      Headers           `json:"headers,omitempty"`
	TimeoutMS    int64             `json:"timeout_ms,omitempty"`
	RetryPolicy  *RetryPolicy      `json:"retry_policy,omitempty"`
	OpaqueConfig map[string]string `json:"opaque_config,omitempty"`

	AutoHostRewrite  bool `json:"auto_host_rewrite,omitempty"`
	WebsocketUpgrade bool `json:"use_websocket,omitempty"`

	// clusters contains the set of referenced clusters in the route; the field is special
	// and used only to aggregate cluster information after composing routes
	clusters ClustersV1

	// faults contains the set of referenced faults in the route; the field is special
	// and used only to aggregate fault filter information after composing routes
	faults []*HTTPFilter
}

// CatchAll returns true if the route matches all requests
func (route *HTTPRoute) CatchAll() bool {
	return len(route.Headers) == 0 && route.Path == "" && route.Prefix == "/"
}

// CombinePathPrefix checks that the route applies for a given path and prefix
// match and updates the path and the prefix in the route. If the route is
// incompatible with the path or the prefix, returns nil.  Either path or
// prefix must be set but not both.  The resulting route must match exactly the
// requests that match both the original route and the supplied path and
// prefix.
func (route *HTTPRoute) CombinePathPrefix(path, prefix string) *HTTPRoute {
	switch {
	case path == "" && route.Path == "" && strings.HasPrefix(route.Prefix, prefix):
		// pick the longest prefix if both are prefix matches
		return route
	case path == "" && route.Path == "" && strings.HasPrefix(prefix, route.Prefix):
		route.Prefix = prefix
		return route
	case prefix == "" && route.Prefix == "" && route.Path == path:
		// pick only if path matches if both are path matches
		return route
	case path == "" && route.Prefix == "" && strings.HasPrefix(route.Path, prefix):
		// if mixed, pick if route path satisfies the prefix
		return route
	case prefix == "" && route.Path == "" && strings.HasPrefix(path, route.Prefix):
		// if mixed, pick if route prefix satisfies the path and change route to path
		route.Path = path
		route.Prefix = ""
		return route
	default:
		return nil
	}
}

// RetryPolicy definition
// See: https://lyft.github.io/envoy/docs/configuration/http_conn_man/route_config/route.html#retry-policy
type RetryPolicy struct {
	Policy          string `json:"retry_on"` //if unset, set to 5xx,connect-failure,refused-stream
	NumRetries      int    `json:"num_retries,omitempty"`
	PerTryTimeoutMS int64  `json:"per_try_timeout_ms,omitempty"`
}

// WeightedCluster definition
// See https://envoyproxy.github.io/envoy/configuration/http_conn_man/route_config/route.html
type WeightedCluster struct {
	ClustersV1         []*WeightedClusterEntry `json:"clusters"`
	RuntimeKeyPrefix string                  `json:"runtime_key_prefix,omitempty"`
}

// WeightedClusterEntry definition. Describes the format of each entry in the WeightedCluster
type WeightedClusterEntry struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

// VirtualHost definition
type VirtualHost struct {
	Name    string       `json:"name"`
	Domains []string     `json:"domains"`
	Routes  []*HTTPRoute `json:"routes"`
}

func (host *VirtualHost) clusters() ClustersV1 {
	out := make(ClustersV1, 0)
	for _, route := range host.Routes {
		out = append(out, route.clusters...)
	}
	return out
}

// HTTPRouteConfig definition
type HTTPRouteConfig struct {
	VirtualHosts []*VirtualHost `json:"virtual_hosts"`
	ValidateClusters bool `json:"validate_clusters,omitempty"`
}

// HTTPRouteConfigs is a map from the port number to the route config
type HTTPRouteConfigs map[int]*HTTPRouteConfig

// EnsurePort creates a route config if necessary
func (routes HTTPRouteConfigs) EnsurePort(port int) *HTTPRouteConfig {
	config, ok := routes[port]
	if !ok {
		config = &HTTPRouteConfig{}
		routes[port] = config
	}
	return config
}

func (routes HTTPRouteConfigs) clusters() ClustersV1 {
	out := make(ClustersV1, 0)
	for _, config := range routes {
		out = append(out, config.clusters()...)
	}
	return out
}

func (routes HTTPRouteConfigs) normalize() HTTPRouteConfigs {
	out := make(HTTPRouteConfigs)

	// sort HTTP routes by virtual hosts, rest should be deterministic
	for port, routeConfig := range routes {
		out[port] = routeConfig.normalize()
	}

	return out
}

// combine creates a new route config that is the union of all HTTP routes.
// note that the virtual hosts without an explicit port suffix (IP:PORT) are stripped
// for all routes except the route for port 80.
func (routes HTTPRouteConfigs) combine() *HTTPRouteConfig {
	out := &HTTPRouteConfig{}
	for port, config := range routes {
		for _, host := range config.VirtualHosts {
			vhost := &VirtualHost{
				Name:   host.Name,
				Routes: host.Routes,
			}
			for _, domain := range host.Domains {
				if port == 80 || strings.Contains(domain, ":") {
					vhost.Domains = append(vhost.Domains, domain)
				}
			}

			if len(vhost.Domains) > 0 {
				out.VirtualHosts = append(out.VirtualHosts, vhost)
			}
		}
	}
	return out.normalize()
}

// faults aggregates fault filters across virtual hosts in single http_conn_man
func (rc *HTTPRouteConfig) faults() []*HTTPFilter {
	out := make([]*HTTPFilter, 0)
	for _, host := range rc.VirtualHosts {
		for _, route := range host.Routes {
			out = append(out, route.faults...)
		}
	}
	return out
}

func (rc *HTTPRouteConfig) clusters() ClustersV1 {
	out := make(ClustersV1, 0)
	for _, host := range rc.VirtualHosts {
		out = append(out, host.clusters()...)
	}
	return out
}

func (rc *HTTPRouteConfig) normalize() *HTTPRouteConfig {
	hosts := make([]*VirtualHost, len(rc.VirtualHosts))
	copy(hosts, rc.VirtualHosts)
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
	return &HTTPRouteConfig{VirtualHosts: hosts}
}

// AccessLog definition.
type AccessLog struct {
	Path   string `json:"path"`
	Format string `json:"format,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// HTTPFilterConfig definition
type HTTPFilterConfig struct {
	CodecType         string                 `json:"codec_type"`
	StatPrefix        string                 `json:"stat_prefix"`
	GenerateRequestID bool                   `json:"generate_request_id,omitempty"`
	UseRemoteAddress  bool                   `json:"use_remote_address,omitempty"`
	Tracing           *HTTPFilterTraceConfig `json:"tracing,omitempty"`
	RouteConfig       *HTTPRouteConfig       `json:"route_config,omitempty"`
	RDS               *RDS                   `json:"rds,omitempty"`
	Filters           []HTTPFilter           `json:"filters"`
	AccessLog         []AccessLog            `json:"access_log"`
}

func (*HTTPFilterConfig) isNetworkFilterConfig() {}

// HTTPFilterTraceConfig definition
type HTTPFilterTraceConfig struct {
	OperationName string `json:"operation_name"`
}

// TCPRoute definition
type TCPRoute struct {
	Cluster           string   `json:"cluster"`
	DestinationIPList []string `json:"destination_ip_list,omitempty"`
	DestinationPorts  string   `json:"destination_ports,omitempty"`
	SourceIPList      []string `json:"source_ip_list,omitempty"`
	SourcePorts       string   `json:"source_ports,omitempty"`

	// special value to retain dependent cluster definition for TCP routes.
	clusterRef *Cluster
}

// TCPRouteByRoute sorts TCP routes over all route sub fields.
type TCPRouteByRoute []*TCPRoute

func (r TCPRouteByRoute) Len() int {
	return len(r)
}

func (r TCPRouteByRoute) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r TCPRouteByRoute) Less(i, j int) bool {
	if r[i].Cluster != r[j].Cluster {
		return r[i].Cluster < r[j].Cluster
	}

	compare := func(a, b []string) bool {
		lenA, lenB := len(a), len(b)
		min := lenA
		if min > lenB {
			min = lenB
		}
		for k := 0; k < min; k++ {
			if a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return lenA < lenB
	}

	if less := compare(r[i].DestinationIPList, r[j].DestinationIPList); less {
		return less
	}
	if r[i].DestinationPorts != r[j].DestinationPorts {
		return r[i].DestinationPorts < r[j].DestinationPorts
	}
	if less := compare(r[i].SourceIPList, r[j].SourceIPList); less {
		return less
	}
	if r[i].SourcePorts != r[j].SourcePorts {
		return r[i].SourcePorts < r[j].SourcePorts
	}
	return false
}

// TCPProxyFilterConfig definition
type TCPProxyFilterConfig struct {
	StatPrefix  string          `json:"stat_prefix"`
	RouteConfig *TCPRouteConfig `json:"route_config"`
}

func (*TCPProxyFilterConfig) isNetworkFilterConfig() {}

// TCPRouteConfig (or generalize as RouteConfig or L4RouteConfig for TCP/UDP?)
type TCPRouteConfig struct {
	Routes []*TCPRoute `json:"routes"`
}

// MONGOProxyFilterConfig definition
type MONGOProxyFilterConfig struct {
	StatPrefix string `json:"stat_prefix"`
	// TODO: support fault filter
}

func (*MONGOProxyFilterConfig) isNetworkFilterConfig() {}

// NetworkFilter definition
type NetworkFilter struct {
	Type                 string          `json:"type"`
	Name                 string          `json:"name"`
	Config               json.RawMessage `json:"config"`
	HTTPFilterConfig     *HTTPFilterConfig
	TCPProxyFilterConfig *TCPProxyFilterConfig
}

// NetworkFilterConfig is a marker interface
type NetworkFilterConfig interface {
	isNetworkFilterConfig()
}

// Listener definition
type Listener struct {
	Address        string           `json:"address"`
	Name           string           `json:"name,omitempty"`
	Filters        []*NetworkFilter `json:"filters"`
	SSLContext     *SSLContext      `json:"ssl_context,omitempty"`
	BindToPort     bool             `json:"bind_to_port"`
	UseOriginalDst bool             `json:"use_original_dst,omitempty"`
	UseProxyProto  bool             `json:"use_proxy_proto,omitempty"`
	PerConnectionBufferLimitBytes int `json:"per_connection_buffer_limit_bytes,omitempty"`
	DrainType      string 			 `json:"drain_type,omitempty"`
}

// Listeners is a collection of listeners
type ListenersV1 []*Listener
type Listeners []*xdsapi.Listener

// normalize sorts and de-duplicates listeners by address
func (listeners ListenersV1) normalize() ListenersV1 {
	out := make(ListenersV1, 0, len(listeners))
	set := make(map[string]bool)
	for _, listener := range listeners {
		if !set[listener.Address] {
			set[listener.Address] = true
			out = append(out, listener)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out
}

// GetByAddress returns a listener by its address
func (listeners ListenersV1) GetByAddress(addr string) *Listener {
	for _, listener := range listeners {
		if listener.Address == addr {
			return listener
		}
	}
	return nil
}

// SSLContext definition
type SSLContext struct {
	CertChainFile            string `json:"cert_chain_file"`
	PrivateKeyFile           string `json:"private_key_file"`
	CaCertFile               string `json:"ca_cert_file,omitempty"`
	RequireClientCertificate bool   `json:"require_client_certificate"`
}

// SSLContextExternal definition
type SSLContextExternal struct {
	CaCertFile string `json:"ca_cert_file,omitempty"`
}

// SSLContextWithSAN definition, VerifySubjectAltName cannot be nil.
type SSLContextWithSAN struct {
	CertChainFile        string   `json:"cert_chain_file"`
	PrivateKeyFile       string   `json:"private_key_file"`
	CaCertFile           string   `json:"ca_cert_file,omitempty"`
	VerifySubjectAltName []string `json:"verify_subject_alt_name"`
}

// Admin definition
type Admin struct {
	AccessLogPath string `json:"access_log_path"`
	Address       string `json:"address"`
}

// Host definition
type Host struct {
	URL string `json:"url"`
}

// Cluster definition
type Cluster struct {
	Name                     string `json:"name"`
	ServiceName              string `json:"service_name,omitempty"`
	ConnectTimeoutMs         int64  `json:"connect_timeout_ms"`
	Type                     string `json:"type"`
	LbType                   string `json:"lb_type"`
	MaxRequestsPerConnection int    `json:"max_requests_per_connection,omitempty"`
	Hosts                    []Host `json:"hosts,omitempty"`
	// SSLContext               interface{}       `json:"ssl_context,omitempty"`
	SSLContext       *SSLContextWithSAN `json:"ssl_context,omitempty"`
	Features         string             `json:"features,omitempty"`
	CircuitBreaker   *CircuitBreaker    `json:"circuit_breakers,omitempty"`
	OutlierDetection *OutlierDetection  `json:"outlier_detection,omitempty"`
	PerConnectionBufferLimitBytes int `json:"per_connection_buffer_limit_bytes,omitempty"`

	// special values used by the post-processing passes for outbound mesh-local clusters
	outbound bool
	hostname string
	port     *Port
	tags     Labels
}

// CircuitBreaker definition
// See: https://lyft.github.io/envoy/docs/configuration/cluster_manager/cluster_circuit_breakers.html#circuit-breakers
type CircuitBreaker struct {
	Default DefaultCBPriority `json:"default"`
	High HighCBPriority `json:"high"`
}

// DefaultCBPriority defines the circuit breaker for default cluster priority
type DefaultCBPriority struct {
	MaxConnections     int `json:"max_connections,omitempty"`
	MaxPendingRequests int `json:"max_pending_requests,omitempty"`
	MaxRequests        int `json:"max_requests,omitempty"`
	MaxRetries         int `json:"max_retries,omitempty"`
}

type HighCBPriority struct {
	MaxConnections     int `json:"max_connections,omitempty"`
	MaxPendingRequests int `json:"max_pending_requests,omitempty"`
	MaxRequests        int `json:"max_requests,omitempty"`
	MaxRetries         int `json:"max_retries,omitempty"`
}

// OutlierDetection definition
// See: https://lyft.github.io/envoy/docs/configuration/cluster_manager/cluster_runtime.html#outlier-detection
type OutlierDetection struct {
	ConsecutiveErrors  int   `json:"consecutive_5xx,omitempty"`
	IntervalMS         int64 `json:"interval_ms,omitempty"`
	BaseEjectionTimeMS int64 `json:"base_ejection_time_ms,omitempty"`
	MaxEjectionPercent int   `json:"max_ejection_percent,omitempty"`
	ConsecutiveGatewayFailure int `json:"consecutive_gateway_failure,omitempty"`
	EnforcingConsecutive5xx int `json:"enforcing_consecutive_5xx,omitempty"`
	EnforcingConsecutiveGatewayFailure int `json:"enforcing_consecutive_gateway_failure,omitempty"`
	EnforcingSuccessRate  int `json:"enforcing_success_rate,omitempty"`
	SuccessRateMinimumHosts int `json:"success_rate_minimum_hosts,omitempty"`
	SuccessRateRequestVolume  int `json:"success_rate_request_volume,omitempty"`
	SuccessRateStdevFactor int `json:"success_rate_stdev_factor,omitempty"`
}

// Clusters is a collection of clusters
type ClustersV1 []*Cluster
type Clusters []*xdsapi.Cluster

// normalize deduplicates and sorts clusters by name
func (clusters ClustersV1) normalize() ClustersV1 {
	out := make(ClustersV1, 0, len(clusters))
	set := make(map[string]bool)
	for _, cluster := range clusters {
		if !set[cluster.Name] {
			set[cluster.Name] = true
			out = append(out, cluster)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// RoutesByPath sorts routes by their path and/or prefix, such that:
// - Exact path routes are "less than" than prefix path routes
// - Exact path routes are sorted lexicographically
// - Prefix path routes are sorted anti-lexicographically
//
// This order ensures that prefix path routes do not shadow more
// specific routes which share the same prefix.
type RoutesByPath []*HTTPRoute

func (r RoutesByPath) Len() int {
	return len(r)
}

func (r RoutesByPath) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RoutesByPath) Less(i, j int) bool {
	if r[i].Path != "" {
		if r[j].Path != "" {
			// i and j are both path
			return r[i].Path < r[j].Path
		}
		// i is path and j is prefix => i is "less than" j
		return true
	}
	if r[j].Path != "" {
		// i is prefix nad j is path => j is "less than" i
		return false
	}
	// i and j are both prefix
	return r[i].Prefix > r[j].Prefix
}

// Headers sorts headers
type Headers []Header

func (s Headers) Len() int {
	return len(s)
}

func (s Headers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Headers) Less(i, j int) bool {
	if s[i].Name == s[j].Name {
		if s[i].Regex == s[j].Regex {
			return s[i].Value < s[j].Value
		}
		// true is less, false is more
		return s[i].Regex
	}
	return s[i].Name < s[j].Name
}

// DiscoveryCluster is a service discovery service definition
type DiscoveryCluster struct {
	Cluster        *Cluster `json:"cluster"`
	RefreshDelayMs int64    `json:"refresh_delay_ms"`
}

// LDSCluster is a reference to LDS cluster by name
type LDSCluster struct {
	Cluster        string `json:"cluster"`
	RefreshDelayMs int64  `json:"refresh_delay_ms"`
}

// RDS definition
type RDS struct {
	Cluster         string `json:"cluster"`
	RouteConfigName string `json:"route_config_name"`
	RefreshDelayMs  int64  `json:"refresh_delay_ms"`
}

// ClusterManager definition
type ClusterManager struct {
	ClustersV1 ClustersV1          `json:"clusters"`
	SDS      *DiscoveryCluster `json:"sds,omitempty"`
	CDS      *DiscoveryCluster `json:"cds,omitempty"`
}

// Port represents a network port where a service is listening for
// connections. The port should be annotated with the type of protocol
// used by the port.
type Port struct {
	// Name ascribes a human readable name for the port object. When a
	// service has multiple ports, the name field is mandatory
	Name string `json:"name,omitempty"`

	// Port number where the service can be reached. Does not necessarily
	// map to the corresponding port numbers for the instances behind the
	// service. See NetworkEndpoint definition below.
	Port int `json:"port"`

	// Protocol to be used for the port.
	Protocol Protocol `json:"protocol,omitempty"`
}

// PortList is a set of ports
type PortList []*Port

// Protocol defines network protocols for ports
type Protocol string

const (
	// ProtocolGRPC declares that the port carries gRPC traffic
	ProtocolGRPC Protocol = "GRPC"
	// ProtocolHTTPS declares that the port carries HTTPS traffic
	ProtocolHTTPS Protocol = "HTTPS"
	// ProtocolHTTP2 declares that the port carries HTTP/2 traffic
	ProtocolHTTP2 Protocol = "HTTP2"
	// ProtocolHTTP declares that the port carries HTTP/1.1 traffic.
	// Note that HTTP/1.0 or earlier may not be supported by the proxy.
	ProtocolHTTP Protocol = "HTTP"
	// ProtocolTCP declares the the port uses TCP.
	// This is the default protocol for a service port.
	ProtocolTCP Protocol = "TCP"
	// ProtocolUDP declares that the port uses UDP.
	// Note that UDP protocol is not currently supported by the proxy.
	ProtocolUDP Protocol = "UDP"
	// ProtocolMONGO declares that the port carries mongoDB traffic
	ProtocolMONGO Protocol = "MONGO"
)

// Labels is a non empty set of arbitrary strings. Each version of a service can
// be differentiated by a unique set of labels associated with the version. These
// labels are assigned to all instances of a particular service version. For
// example, lets say catalog.mystore.com has 2 versions v1 and v2. v1 instances
// could have labels gitCommit=aeiou234, region=us-east, while v2 instances could
// have labels name=kittyCat,region=us-east.
type Labels map[string]string

type ListenersData struct {
	ListenersV1 ListenersV1 `json:"listeners"`
}

type ServiceHosts struct {
	Hosts []*ServiceHost `json:"hosts"`
}

type ServiceHost struct {
	Address string `json:"ip_address"`
	Port    int    `json:"port"`
	Tags    *Tags  `json:"tags,omitempty"`
}

type Tags struct {
	AZ     string `json:"az,omitempty"`
	Canary bool   `json:"canary,omitempty"`

	// Weight is an integer in the range [1, 100] or empty
	Weight int `json:"load_balancing_weight,omitempty"`
}

type KeyAndService struct {
	Key   string         `json:"service-key"`
	Hosts []*ServiceHost `json:"hosts"`
}

type BytesValue struct {
	BytesValue	string	`json:"bytesValue,omitempty"`
}



type StringValue struct {
	StringValue	string	`json:"stringValue,omitempty"`
}




type Attributes struct {
	SourceIp 		*BytesValue 	`json:"source.ip,omitempty"`
	SourceUid 		*StringValue 	`json:"source.uid,omitempty"`
	DestinationIp 	*BytesValue 	`json:"destination.ip,omitempty"`
	DestinationUid *StringValue 	`json:"destination.uid,omitempty"`
	DestinationService *StringValue 	`json:"destination.service,omitempty"`
}

type AttributeConfig struct {
	Attributes *Attributes `json:"attributes,omitempty"`
}


// FilterMixerConfig definition
type FilterMixerV2Config struct {

	V2  *FilterMixerConfig `json:"v2"`
}

type FilterMixerConfig struct {
	// MixerAttributes specifies the static list of attributes that are sent with
	// each request to Mixer.
	MixerAttributes *AttributeConfig `json:"mixerAttributes"`

	// ForwardAttributes specifies the list of attribute keys and values that
	// are forwarded as an HTTP header to the server side proxy
	ForwardAttributes *AttributeConfig `json:"forwardAttributes"`

	// QuotaName specifies the name of the quota bucket to withdraw tokens from;
	// an empty name means no quota will be charged.
	QuotaName string `json:"quota_name,omitempty"`

	// If set to true, disables mixer check calls for TCP connections
	DisableTCPCheckCalls bool `json:"disable_tcp_check_calls,omitempty"`
}