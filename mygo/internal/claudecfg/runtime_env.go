package claudecfg

import (
	"strconv"
	"strings"

	"github.com/ixxxxoooo/minicode/internal/core"
)

// RuntimeEnv 根据 MiniCode 运行环境偏好生成子进程环境变量(不记录值,只写变量名)。
func RuntimeEnv(proxyMode int, proxyURL, noProxy, timeoutSec string) []core.EnvVar {
	var out []core.EnvVar
	switch proxyMode {
	case 1: // 直连:清空常见代理变量,避免继承 shell
		for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
			out = append(out, core.EnvVar{Key: k, Value: ""})
		}
	case 2: // 自定义
		u := strings.TrimSpace(proxyURL)
		if u != "" {
			for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
				out = append(out, core.EnvVar{Key: k, Value: u})
			}
		}
		np := strings.TrimSpace(noProxy)
		if np != "" {
			out = append(out, core.EnvVar{Key: "NO_PROXY", Value: np})
			out = append(out, core.EnvVar{Key: "no_proxy", Value: np})
		}
	}
	timeoutSec = strings.TrimSpace(timeoutSec)
	if timeoutSec != "" {
		if sec, err := strconv.Atoi(timeoutSec); err == nil && sec > 0 {
			out = append(out, core.EnvVar{Key: "API_TIMEOUT_MS", Value: strconv.Itoa(sec * 1000)})
		}
	}
	return out
}
