package utils

import (
	"net/url"

	"github.com/Is999/go-utils/errors"
)

// - 对URL字符转义 - url.QueryEscape(str)
// - 对URL转义字符反转义 - url.QueryUnescape(str)

// UrlPath 组装带参数的url
func UrlPath(urlPath string, params url.Values) (string, error) {
	if params != nil && len(params) > 0 {
		Url, err := url.Parse(urlPath)
		if err != nil {
			return "", errors.Wrap(err)
		}
		query := Url.Query()
		for key, val := range params {
			query[key] = val
		}
		Url.RawQuery = query.Encode()
		return Url.String(), nil
	}
	return urlPath, nil
}
