// tools/stock.go
// -----------------------------------------------------------------------------
// 【这是什么】股票工具。重点演示"服务端限流 / 429 响应"的处理姿势。
// 【演示什么】
//   1. 调 errors.ClassifyHTTPError 把 HTTP 状态码翻译成 Temporal 错误
//   2. 遇到 429 + Retry-After 时,把"下次重试时间"透给 Temporal,
//      让平台按服务器的要求 backoff,而不是瞎猜指数退避
// 【类比】排队取号。服务员喊"30 分钟后再来",你就准时 30 分钟后来,
//   别一直在窗口前问个没完,否则迟早被拉黑。
// -----------------------------------------------------------------------------

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apperrors "github.com/CXP-shawn/temporal-ai-agent-tutorial-go/errors"
)

type StockHandler struct{}

type stockArgs struct {
	Symbol string `json:"symbol"`
}

type stockResult struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

func (StockHandler) Name() string        { return "stock" }
func (StockHandler) Description() string { return "查询指定股票的最新价格" }

func (StockHandler) Validate(args json.RawMessage) error {
	var a stockArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}
	if a.Symbol == "" {
		return fmt.Errorf("symbol 不能为空")
	}
	return nil
}

// simulateCount 用来演示"第一次请求被 429,第二次成功"。
// 生产代码不要这样写全局变量;这里只为教学。
var simulateCount int

func (StockHandler) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a stockArgs
	_ = json.Unmarshal(args, &a)

	simulateCount++
	if simulateCount == 1 {
		// 第一次调用,模拟服务端返回 429 + Retry-After: 2
		h := http.Header{}
		h.Set("Retry-After", "2")
		body := []byte(`{"error":"rate_limited"}`)
		return nil, apperrors.ClassifyHTTPError(http.StatusTooManyRequests, h, body)
	}

	// 第二次及以后 —— 正常返回
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return json.Marshal(stockResult{Symbol: a.Symbol, Price: 258.42})
}

func (StockHandler) TimeoutHint() time.Duration { return 30 * time.Second }
func (StockHandler) IsIdempotent() bool         { return true } // 查价格是只读
