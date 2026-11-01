package utils

import "regexp"

var sensitiveParameterName = regexp.MustCompile(`(?i)(password|passwd|secret|token|credential|private_key)`)

// SensitiveParameterName 统一模板参数、发布预览和脚本日志的敏感名称识别。
func SensitiveParameterName(name string) bool { return sensitiveParameterName.MatchString(name) }
