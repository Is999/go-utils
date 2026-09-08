package utils

// Base64 直接使用 encoding/base64：StdEncoding 用于普通文本，URLEncoding 用于 URL 或文件名。
// 两者默认保留 = 填充；无填充协议使用对应的 RawStdEncoding 或 RawURLEncoding。
