// errors/classify.go
// -----------------------------------------------------------------------------
// 【这是什么】把 HTTP 的各种状态码翻译成 Temporal 能理解的错误类型。
// 【为什么】Temporal 的 RetryPolicy 靠"错误类型名 + 是否 NonRetryable"来决定重试策略。
//   如果工具只返回一个裸 error,Temporal 只能"无脑重试",浪费上游配额。
//   把状态码分类后:
//     - 429 带 Retry-After → 可重试,但等指定时间再来
//     - 401/403           → 不可重试(AUTH_ERROR)
//     - 4xx               → 不可重试(CLIENT_ERROR)
//     - 5xx               → 可重试(普通 error,走默认 backoff)
// 【在哪层】底层工具包,所有工具文件都可以引用它。
// -----------------------------------------------------------------------------

package errors

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
)

// ClassifyHTTPError 把一次 HTTP 响应(status + headers + body)翻译成合适的 Go error。
//
// 返回值约定:
//   - nil:表示没错误(调用方自己判断是否应该调用本函数)
//   - *ApplicationError(NonRetryable=true):永远别重试
//   - *ApplicationError(带 NextRetryDelay):按指定间隔再试
//   - 普通 error:按 RetryPolicy 默认重试
func ClassifyHTTPError(status int, headers http.Header, body []byte) error {
	switch {
	case status == 429:
		// 429 Too Many Requests:尊重 Retry-After
		delay := parseRetryAfterHeader(headers.Get("Retry-After"))
		return temporal.NewApplicationErrorWithOptions(
			fmt.Sprintf("上游限流 (429),建议等待 %s", delay),
			"RATE_LIMITED",
			temporal.ApplicationErrorOptions{
				NonRetryable:   false,
				NextRetryDelay: delay,
			},
		)

	case status == 401 || status == 403:
		// 认证/权限错误,重试 100 次也还是错
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("认证失败 (%d): %s", status, safeBody(body)),
			"AUTH_ERROR",
			nil,
		)

	case status >= 400 && status < 500:
		// 其它 4xx:大多是客户端自己的问题,重试无意义
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("客户端错误 (%d): %s", status, safeBody(body)),
			"CLIENT_ERROR",
			nil,
		)

	case status >= 500:
		// 5xx:服务端抽风,值得重试。返回普通 error,让 Temporal 走默认 RetryPolicy。
		return fmt.Errorf("服务端错误 (%d): %s", status, safeBody(body))
	}
	return nil
}

// parseRetryAfterHeader 解析 Retry-After。
// 它可能是 "120"(秒数),也可能是 HTTP-date。我们这里只处理秒数,别的给个兜底值。
func parseRetryAfterHeader(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 5 * time.Second
	}
	// 数字:秒
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	// HTTP-date:尝试解析成绝对时间,再算差值
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d
	}
	return 5 * time.Second
}

// safeBody 把二进制 body 截短成前 200 字节,避免日志里塞一坨 HTML。
func safeBody(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}
