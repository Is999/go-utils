package utils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"

	"github.com/Is999/go-utils/errors"
)

// initTransport 首次发送或代理/TLS 配置变化时重建 Transport，其余请求复用连接池。
func (c *Curl) initTransport() error {
	if c.cli.Transport != nil {
		if !c.transportDirty {
			return nil
		}
		// 关闭旧池的空闲连接，不中断正在使用的连接。
		c.cli.CloseIdleConnections()
	}

	if c.defLogOutput {
		c.Logger.Debug("Init Transport")
	}

	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok || tr == nil {
		return errors.New("http.DefaultTransport 类型异常")
	}
	// Clone 复制默认配置和 TLS 配置，连接池由当前 Curl 独立持有。
	tr = tr.Clone()

	// 仅在显式设置代理时覆盖默认 Transport 的代理规则。
	if c.proxyURL != "" {
		if c.defLogOutput {
			c.Logger.Debug("ProxyURL()")
		}
		if err := ProxyURL(tr, c.proxyURL); err != nil {
			return err
		}
	}
	if err := c.applyTransportTLS(tr); err != nil {
		return err
	}

	// 全部配置成功后再替换，失败时保留原实例并允许下次重试初始化。
	c.cli.Transport = tr
	c.transportDirty = false
	return nil
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
			return err
		}
	}
	if c.cert != "" && c.key != "" {
		if c.defLogOutput {
			c.Logger.Debug("Certificate()")
		}
		if err := Certificate(ensureTLSConfig(tr), c.cert, c.key); err != nil {
			return err
		}
	}
	return nil
}

// ensureTLSConfig 复用已有 TLS 配置；仅在缺失时创建最低 TLS 1.2 的配置。
func ensureTLSConfig(transport *http.Transport) *tls.Config {
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return transport.TLSClientConfig
}

// ProxyURL 用指定代理替换 Transport 的代理选择函数；调用方应在开始发送前配置。
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

// RootCAs 用系统根证书加指定 PEM 文件构成的证书池替换 config.RootCAs。
// 系统证书池不可用时只使用文件中的证书；读取或解析失败时保留原配置。
func RootCAs(config *tls.Config, rootCAs string) error {
	if config == nil {
		return errors.New("tls.Config 不能为空")
	}
	cert, err := os.ReadFile(rootCAs)
	if err != nil {
		return errors.Tag(err)
	}

	// 不依赖调用方已有的 RootCAs，重复配置不会累积旧文件中的证书。
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

// Certificate 加载证书与私钥，成功后替换客户端证书列表；失败时保留原配置。
func Certificate(config *tls.Config, certFile, keyFile string) error {
	if config == nil {
		return errors.New("tls.Config 不能为空")
	}
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return errors.Tag(err)
	}

	config.Certificates = []tls.Certificate{certificate}
	return nil
}
