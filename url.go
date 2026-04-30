package utils

import (
	"net/url"

	"github.com/Is999/go-utils/errors"
)

// - 对URL字符转义 - url.QueryEscape(str)
// - 对URL转义字符反转义 - url.QueryUnescape(str)

// UrlPath 组装带参数的完整 URL。
// 将 params 中的查询参数合并到 urlPath 中，保留原有查询参数。
//
// 参数说明：
//   - urlPath：基础 URL 路径
//   - params：查询参数键值对
//
// 返回值：完整 URL 字符串，错误信息
func UrlPath(urlPath string, params url.Values) (string, error) {
	if params != nil && len(params) > 0 {
		u, err := url.Parse(urlPath)
		if err != nil {
			return "", errors.Wrap(err)
		}
		query := u.Query()
		for key, val := range params {
			query[key] = val
		}
		u.RawQuery = query.Encode()
		return u.String(), nil
	}
	return urlPath, nil
}
