package utils

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"os"

	apperrors "github.com/Is999/go-utils/errors"
)

// ============================ Transport 初始化 ============================

// initTransport 初始化 HTTP Transport。
// 配置代理、TLS 证书、不安全验证等传输层选项。
//
// 返回值：错误信息
func (c *Curl) initTransport() error {
	// 确保 Client 已初始化
	if c.cli == nil {
		c.cli = &http.Client{}
	}

	// 已有 Transport 则跳过
	if c.cli.Transport != nil {
		return nil
	}

	// Debug 日志
	if c.defLogOutput {
		c.Logger.Debug("Init Transport")
	}

	// 基于标准库默认 Transport 克隆，保留连接池、HTTP/2、代理和超时等生产默认值。
	tr, err := defaultHTTPTransport()
	if err != nil {
		return apperrors.Wrap(err)
	}

	// 配置代理
	if len(c.proxyURL) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("ProxyURL()")
		}
		if err := ProxyURL(tr, c.proxyURL); err != nil {
			return apperrors.Wrap(err)
		}
	}

	// 配置 HTTPS 不安全验证
	if c.insecureSkipVerify {
		if c.defLogOutput {
			c.Logger.Debug("InsecureSkipVerify")
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = defaultTLSConfig()
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	// 配置根证书
	if len(c.rootCAs) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("RootCAs()")
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = defaultTLSConfig()
		}
		if err := RootCAs(tr.TLSClientConfig, c.rootCAs); err != nil {
			return apperrors.Wrap(err)
		}
	}

	// 配置客户端证书
	if len(c.cert) > 0 && len(c.key) > 0 {
		if c.defLogOutput {
			c.Logger.Debug("Certificate()")
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = defaultTLSConfig()
		}
		if err := Certificate(tr.TLSClientConfig, c.cert, c.key); err != nil {
			return apperrors.Wrap(err)
		}
	}

	// 设置 Transport
	c.cli.Transport = tr
	return nil
}

// defaultHTTPTransport 返回标准库默认 Transport 的可修改副本。
//
// 返回值：HTTP Transport、错误信息。
func defaultHTTPTransport() (*http.Transport, error) {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok || tr == nil {
		return nil, apperrors.New("http.DefaultTransport 类型异常")
	}
	cloned := tr.Clone()
	if cloned.TLSClientConfig != nil {
		cloned.TLSClientConfig = cloned.TLSClientConfig.Clone()
	}
	return cloned, nil
}

// defaultTLSConfig 返回生产默认 TLS 配置。
//
// 返回值：TLS 配置指针。
func defaultTLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// ============================ 证书配置函数 ============================

// ProxyURL 设置 HTTP 代理。
//
// 参数说明：
//   - transport：HTTP Transport
//   - proxyURL：代理地址
//
// 返回值：错误信息
func ProxyURL(transport *http.Transport, proxyURL string) error {
	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return apperrors.Wrap(err)
	}
	transport.Proxy = http.ProxyURL(proxy)
	return nil
}

// RootCAs 设置根证书池。
//
// 参数说明：
//   - config：TLS 配置
//   - rootCAs：根证书文件路径
//
// 返回值：错误信息
func RootCAs(config *tls.Config, rootCAs string) error {
	// 读取根证书文件
	cert, err := os.ReadFile(rootCAs)
	if err != nil {
		return apperrors.Wrap(err)
	}

	// 创建证书池
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(cert); !ok {
		return apperrors.Errorf("RootCAs() 未解析到有效 PEM 证书: %s", rootCAs)
	}

	config.RootCAs = certPool
	return nil
}

// Certificate 设置客户端证书。
//
// 参数说明：
//   - config：TLS 配置
//   - certFile：证书文件路径
//   - keyFile：私钥文件路径
//
// 返回值：错误信息
func Certificate(config *tls.Config, certFile, keyFile string) error {
	// 加载客户端证书
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return apperrors.Wrap(err)
	}

	config.Certificates = []tls.Certificate{certificate}
	return nil
}
