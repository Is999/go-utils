package utils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"

	"github.com/Is999/go-utils/errors"
)

// ============================ Transport 初始化 ============================

// initTransport 初始化 HTTP Transport。
// 配置代理、TLS 证书、不安全验证等传输层选项。
func (c *Curl) initTransport() error {
	if c.cli.Transport != nil {
		// 传输层配置未发生变化时，直接复用现有 Transport 与连接池。
		if !c.transportDirty {
			return nil
		}
		// 仅在代理/TLS 配置变化时重建 Transport，避免每次请求都丢失连接复用收益。
		c.cli.CloseIdleConnections()
	}

	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("Init Transport")
	}

	// 基于标准库默认 Transport 克隆，保留连接池、HTTP/2、代理和超时等生产默认值。
	tr, err := defaultHTTPTransport()
	if err != nil {
		return errors.Tag(err)
	}

	if err = c.applyTransportProxy(tr); err != nil {
		return errors.Tag(err)
	}
	if err = c.applyTransportTLS(tr); err != nil {
		return errors.Tag(err)
	}

	// 设置 Transport
	c.cli.Transport = tr
	c.transportDirty = false
	return nil
}

// applyTransportProxy 应用 HTTP 代理配置。
func (c *Curl) applyTransportProxy(tr *http.Transport) error {
	if c.proxyURL == "" {
		return nil
	}
	if c.defLogOutput {
		c.Logger.Debug("ProxyURL()")
	}
	return ProxyURL(tr, c.proxyURL)
}

// applyTransportTLS 应用 TLS 验证、根证书和客户端证书配置。
func (c *Curl) applyTransportTLS(tr *http.Transport) error {
	if c.insecureSkipVerify {
		if c.defLogOutput {
			c.Logger.Debug("InsecureSkipVerify")
		}
		ensureTLSConfig(tr).InsecureSkipVerify = true
	}
	if c.rootCAs != "" {
		if c.defLogOutput {
			c.Logger.Debug("RootCAs()")
		}
		if err := RootCAs(ensureTLSConfig(tr), c.rootCAs); err != nil {
			return errors.Tag(err)
		}
	}
	if c.cert != "" && c.key != "" {
		if c.defLogOutput {
			c.Logger.Debug("Certificate()")
		}
		if err := Certificate(ensureTLSConfig(tr), c.cert, c.key); err != nil {
			return errors.Tag(err)
		}
	}
	return nil
}

// defaultHTTPTransport 返回标准库默认 Transport 的可修改副本。
func defaultHTTPTransport() (*http.Transport, error) {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok || tr == nil {
		return nil, errors.New("http.DefaultTransport 类型异常")
	}
	cloned := tr.Clone()
	if cloned.TLSClientConfig != nil {
		cloned.TLSClientConfig = cloned.TLSClientConfig.Clone()
	}
	return cloned, nil
}

// ensureTLSConfig 返回可写 TLS 配置，缺失时使用生产默认配置。
func ensureTLSConfig(transport *http.Transport) *tls.Config {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return transport.TLSClientConfig
}

// ============================ 证书配置函数 ============================

// ProxyURL 设置 HTTP 代理。
func ProxyURL(transport *http.Transport, proxyURL string) error {
	if transport == nil {
		return errors.New("http.Transport 不能为空")
	}
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return errors.Tag(err)
	}
	transport.Proxy = http.ProxyURL(proxy)
	return nil
}

// RootCAs 设置根证书池。
// 默认会在系统根证书池基础上追加自定义根证书，避免误把系统根证书整体替换掉。
func RootCAs(config *tls.Config, rootCAs string) error {
	if config == nil {
		return errors.New("tls.Config 不能为空")
	}
	// 读取根证书文件
	cert, err := os.ReadFile(rootCAs)
	if err != nil {
		return errors.Tag(err)
	}

	// 优先复用系统根证书池，兼容既有公网证书链。
	certPool, err := x509.SystemCertPool()
	if err != nil || certPool == nil {
		certPool = x509.NewCertPool()
	}
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		return errors.Errorf("RootCAs() 未解析到有效 PEM 证书: %s", rootCAs)
	}

	config.RootCAs = certPool
	return nil
}

// Certificate 设置客户端证书。
func Certificate(config *tls.Config, certFile, keyFile string) error {
	if config == nil {
		return errors.New("tls.Config 不能为空")
	}
	// 加载客户端证书
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return errors.Tag(err)
	}

	config.Certificates = []tls.Certificate{certificate}
	return nil
}
