package utils

import (
	"maps"
	"net/url"

	"github.com/Is999/go-utils/errors"
)

// URLPath 将查询参数合并到 URL，同名键由 params 整组覆盖，其余原参数保留。
// params 为空时原样返回且不解析 URL；非空时按键排序编码，不修改 params。
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
