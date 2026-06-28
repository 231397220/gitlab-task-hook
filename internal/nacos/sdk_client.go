package nacos

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"gitlab-task-hook/internal/bootstrap"
)

// SDKClient wraps nacos-sdk-go/v2's IConfigClient and pre-binds dataID/group/namespace.
type SDKClient struct {
	inner       config_client.IConfigClient
	dataID      string
	group       string
	namespaceID string
}

// NewSDKClient creates a Nacos config client from bootstrap config.
func NewSDKClient(cfg bootstrap.NacosConf) (*SDKClient, error) {
	host, port, err := parseServerAddr(cfg.ServerAddr)
	if err != nil {
		return nil, err
	}

	timeoutMs := uint64(5000)
	if cfg.TimeoutSeconds > 0 {
		timeoutMs = uint64(cfg.TimeoutSeconds) * 1000
	}

	clientCfg := constant.ClientConfig{
		NamespaceId:         cfg.NamespaceID,
		TimeoutMs:           timeoutMs,
		Username:            cfg.Username,
		Password:            cfg.Password,
		AccessKey:           cfg.AccessKey,
		SecretKey:           cfg.SecretKey,
		LogDir:              "/tmp/nacos-sdk/log",
		CacheDir:            "/tmp/nacos-sdk/cache",
		LogLevel:            "error", // suppress SDK internal logs
		NotLoadCacheAtStart: true,
	}

	serverCfgs := []constant.ServerConfig{
		{
			Scheme:      "http",
			IpAddr:      host,
			Port:        port,
			ContextPath: "/nacos",
		},
	}

	inner, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &clientCfg,
		ServerConfigs: serverCfgs,
	})
	if err != nil {
		return nil, fmt.Errorf("create nacos config client: %w", err)
	}

	return &SDKClient{
		inner:       inner,
		dataID:      cfg.DataID,
		group:       cfg.Group,
		namespaceID: cfg.NamespaceID,
	}, nil
}

func (c *SDKClient) GetConfig() (string, error) {
	return c.inner.GetConfig(vo.ConfigParam{
		DataId: c.dataID,
		Group:  c.group,
	})
}

func (c *SDKClient) ListenConfig(onChange func(string)) error {
	return c.inner.ListenConfig(vo.ConfigParam{
		DataId: c.dataID,
		Group:  c.group,
		OnChange: func(_, _, _, data string) {
			onChange(data)
		},
	})
}

func (c *SDKClient) CancelListenConfig() error {
	return c.inner.CancelListenConfig(vo.ConfigParam{
		DataId: c.dataID,
		Group:  c.group,
	})
}

// parseServerAddr splits "http://host:port" into host and port number.
func parseServerAddr(addr string) (host string, port uint64, err error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid server_addr %q: %w", addr, err)
	}
	host = u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "8848"
	}
	port, err = strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in server_addr %q: %w", addr, err)
	}
	return host, port, nil
}
