package nacos

// NacosClient abstracts Nacos config operations so the syncer can be unit-tested
// with a mock without a real Nacos server.
type NacosClient interface {
	// GetConfig fetches the current config content from Nacos.
	GetConfig() (string, error)

	// ListenConfig registers an async callback that is invoked whenever the config
	// changes on the Nacos server. The call returns immediately; the SDK drives
	// long-polling internally. Returns an error if the listener cannot be registered.
	ListenConfig(onChange func(data string)) error

	// CancelListenConfig stops the registered listener and releases SDK resources.
	CancelListenConfig() error
}
