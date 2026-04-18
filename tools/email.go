// tools/email.go
// -----------------------------------------------------------------------------
// 【这是什么】邮件工具。重点演示"不幂等操作的幂等化"。
// 【为什么重要】
//   Temporal 允许 Activity 在失败时自动重试。如果"发邮件"本身就不幂等,
//   重试就会导致收件人收到两封、三封甚至八封一样的邮件 —— 这是事故。
//   解决办法:用 activity.GetInfo().ActivityID 作为"幂等键",
//   把"本次已发送"记下来;重试进来时,一看键已存在,直接返回"已发送"即可。
// 【类比】快递员送货前先看"单号":已经签收过,就不再按第二次门铃。
// -----------------------------------------------------------------------------

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
)

type EmailHandler struct{}

type emailArgs struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type emailResult struct {
	Status string `json:"status"` // "sent" 或 "already sent"
	To     string `json:"to"`
}

// sentCache 模拟"发送记录"。真实项目请换成 Redis / 数据库唯一索引。
var sentCache sync.Map // key: activityID, value: true

func (EmailHandler) Name() string        { return "email" }
func (EmailHandler) Description() string { return "给指定邮箱发送一封邮件" }

func (EmailHandler) Validate(args json.RawMessage) error {
	var a emailArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return fmt.Errorf("解析参数失败: %w", err)
	}
	if a.To == "" {
		return fmt.Errorf("to 不能为空")
	}
	if a.Body == "" {
		return fmt.Errorf("body 不能为空")
	}
	return nil
}

func (EmailHandler) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a emailArgs
	_ = json.Unmarshal(args, &a)

	// Temporal 为每一次 Activity 尝试分配稳定的 ActivityID(同一 Activity
	// 的多次重试共享同一个 ID),非常适合当幂等键。
	info := activity.GetInfo(ctx)
	idempotencyKey := info.ActivityID

	if _, loaded := sentCache.LoadOrStore(idempotencyKey, true); loaded {
		// 已经发过了,直接返回 —— 这就是"幂等"的精髓
		return json.Marshal(emailResult{Status: "already sent", To: a.To})
	}

	// 模拟发邮件耗时
	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return json.Marshal(emailResult{Status: "sent", To: a.To})
}

func (EmailHandler) TimeoutHint() time.Duration { return 60 * time.Second }
func (EmailHandler) IsIdempotent() bool         { return false } // 天生不幂等 —— 所以要自己做去重
