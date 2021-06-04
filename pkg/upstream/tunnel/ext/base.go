package ext

type ServerLister interface {
	List(string) (chan []string)
}

var serverListers = map[string]ServerLister{}

func RegisterServerLister(name string, lister ServerLister) {
	serverListers[name] = lister
}

func GetServerLister(name string) ServerLister {
	return serverListers[name]
}

type ConnectionCredential interface {
	CreateCredential(cluster string) string
	ValidateCredential(credential string, host string, cluster string) bool
}

func RegisterConnectionCredential(name string, lister ServerLister) {
	serverListers[name] = lister
}

func GetConnectionCredential(name string) ConnectionCredential {
	return connectionCredentials[name]
}

var connectionCredentials = map[string]ConnectionCredential{}
