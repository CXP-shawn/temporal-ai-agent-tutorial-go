// llm/fake_llm.go
// -----------------------------------------------------------------------------
// 【这是什么】一个用脚本冒充 LLM 的假模型。
// 【为什么】教学不想让你去办 API Key,所以用一段写死的"下一步决策"脚本。
//   真实项目里,你会把"历史对话"发给 OpenAI/Claude,解析它返回的 function_call,
//   最终产出一个 Decision 对象 —— 跟这里的接口一模一样。
// -----------------------------------------------------------------------------

package llm

// Decision 描述 LLM 每一轮的输出:要么调工具,要么说"我干完了"。
type Decision struct {
	Action   string                 // "call_tool" 或 "done"
	ToolName string                 // Action=call_tool 时用
	Args     map[string]interface{} // 工具参数
	Summary  string                 // Action=done 时的最终答复
}

// FakeLLM 是一个按轮次返回脚本的"模型"。
type FakeLLM struct{}

// NewFakeLLM 构造函数。
func NewFakeLLM() *FakeLLM { return &FakeLLM{} }

// NextDecision 按当前轮次返回下一步决策。
//
// 演示脚本:
//   round 0 → 查天气
//   round 1 → 查股价
//   round 2 → 发邮件汇总
//   round 3 → 收工
func (FakeLLM) NextDecision(round int) Decision {
	switch round {
	case 0:
		return Decision{
			Action:   "call_tool",
			ToolName: "weather",
			Args:     map[string]interface{}{"city": "Shanghai"},
		}
	case 1:
		return Decision{
			Action:   "call_tool",
			ToolName: "stock",
			Args:     map[string]interface{}{"symbol": "TSLA"},
		}
	case 2:
		return Decision{
			Action:   "call_tool",
			ToolName: "email",
			Args: map[string]interface{}{
				"to":   "user@example.com",
				"body": "今日上海天气晴转多云 22°C,TSLA 最新价 258.42。",
			},
		}
	default:
		return Decision{
			Action:  "done",
			Summary: "今日简报已发送",
		}
	}
}
