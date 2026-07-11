package utils

import (
	"maps"
	"net/url"

	"github.com/Is999/go-utils/errors"
)

// URLPath 组装带参数的完整 URL。
// 将 params 中的查询参数合并到 urlPath 中，保留原有查询参数。
func URLPath(urlPath string, params url.Values) (string, error) {
	if len(params) == 0 {
		return urlPath, nil
	}
	u, err := url.Parse(urlPath)
	if err != nil {
		return "", errors.Tag(err)
	}
	query := u.Query()
	maps.Copy(query, params)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
