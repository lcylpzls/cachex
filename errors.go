package cachex

import "github.com/lcylpzls/errx"

// 错误码定义:cachex 各失败场景的错误码。
const (
	// CodeInvalidConfig 配置非法。
	CodeInvalidConfig errx.Code = "CACHEX_INVALID_CONFIG"
)

func init() {
	errx.RegisterCode(CodeInvalidConfig, "配置非法")
}
