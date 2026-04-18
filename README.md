# Temporal AI Agent 教学仓库 (Go)

> 面向小白的 Go + Temporal AI Agent 最小可运行示例。每个文件都有逐行中文注释。

## 一、三个核心概念(生活化类比)

想象你要拍一部电影:

- **Workflow(剧本)**: 规定"先干啥、再干啥、遇到意外怎么办"。剧本本身不动手,它只指挥。它必须是 **确定性** 的 —— 同样的剧本,重播多少次结果都一样。
- **Activity(演员)**: 真正去做事的人。发 HTTP 请求、调 LLM、查数据库、发邮件 —— 这些"和外部世界打交道"的脏活,统统交给演员。
- **Worker(剧团)**: 把剧本和演员聚到一个地方、连上 Temporal Server、等着被派活。没有 Worker,剧本只是躺在抽屉里的一张纸。

再加一个关键角色:

- **Temporal Server(制片厂 + 档案室)**: 记录每一步"谁上场、说了什么、结果是什么"。哪怕机器宕机重启,Server 回放历史就能让 Workflow 从中断的地方继续。

## 二、架构图

```
        ┌─────────────┐
        │  FakeLLM    │  ← 假装是 OpenAI,给出"下一步干啥"
        └──────┬──────┘
               │ Decision
               ▼
        ┌─────────────────────┐
        │  AgentWorkflow      │  ← 剧本:循环问 LLM、调工具、直到 done
        │  (workflows/)       │
        └──────┬──────────────┘
               │ ExecuteActivity
               ▼
        ┌─────────────────────┐
        │ ExecuteToolActivity │  ← 演员:限流 → 查注册表 → 真正执行
        │  (activities/)      │
        └──────┬──────────────┘
               │
               ▼
        ┌─────────────────────┐
        │   ToolRegistry      │  ← 菜单:按名字找到对应 handler
        │   (tools/)          │
        └──────┬──────────────┘
               │
        ┌──────┴──────┬─────────────┐
        ▼             ▼             ▼
     Weather        Stock         Email
     Handler        Handler       Handler
```

## 三、文件作用一句话表

| 文件 | 作用 |
|---|---|
| `main.go` | 启动 Worker、注册 Workflow/Activity、触发一次演示 |
| `workflows/agent_workflow.go` | 剧本本体:Agentic Loop(问 LLM → 调工具 → 重复) |
| `activities/tool_activity.go` | 演员入口:限流 + 调 ToolRegistry |
| `tools/types.go` | 工具统一接口(菜单格式定义) |
| `tools/registry.go` | 工具注册表(菜单本身) |
| `tools/weather.go` | 天气工具(演示幂等只读) |
| `tools/stock.go` | 股票工具(演示 429/Retry-After 错误分类) |
| `tools/email.go` | 邮件工具(演示发邮件幂等去重) |
| `llm/fake_llm.go` | 假 LLM:按脚本返回"下一步" |
| `errors/classify.go` | HTTP 错误分类 → Temporal 可重试/不可重试错误 |
| `codec/claim_check.go` | 大 payload 外置(行李寄存牌模式) |
| `ratelimit/user_limiter.go` | 按 userID 的并发上限 |
| `go.mod` | Go 模块声明 |

## 四、推荐阅读顺序

1. `tools/types.go` — 先看"工具长什么样"
2. `tools/weather.go` — 看一个具体工具
3. `tools/registry.go` — 看工具怎么被登记
4. `activities/tool_activity.go` — 看谁去调工具
5. `workflows/agent_workflow.go` — 看剧本怎么循环
6. `main.go` — 看一切怎么串起来

## 五、运行步骤

```bash
# 1. 安装 Temporal CLI(macOS)
brew install temporal

# 2. 启动本地开发 Server(另开一个终端,保持运行)
temporal server start-dev
# Web UI: http://localhost:8233

# 3. 下载依赖
go mod tidy

# 4. 跑起来
go run main.go
```

看到打印 `今日简报已发送` 就说明整个 Agentic Loop 跑通了。

## 六、想改点啥?

- 换真 LLM: 替换 `llm/fake_llm.go` 里的 `NextDecision`,接 OpenAI/Claude API
- 加新工具: 在 `tools/` 下新建文件,实现 `ToolHandler` 接口,在 `registry.go` 的 `init()` 里 `Register(...)`
- 调限流: `ratelimit/user_limiter.go` 改 `MaxPerUser`

祝玩得开心。
