package utils

// HTML 实体转换直接使用 html.EscapeString 和 html.UnescapeString。
// 这两个函数只处理实体编码，不负责 HTML 标签过滤。
