package internal

// ProxyInfo is represent the redis proxy instance is online or not.
type ProxyInfo struct {
	Addr  string `json:"addr"`
	State string `json:"state"`
}
