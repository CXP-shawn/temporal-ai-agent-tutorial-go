// main.go
// -----------------------------------------------------------------------------
// 【这是什么】整个教学项目的入口。
// 【在哪层】最外面一层。它不参与剧本也不扮演演员,它只负责"开场前的准备":
//   1. 连上 Temporal Server(localhost:7233)
//   2. 起一个 Worker(剧团),把 Workflow(剧本)和 Activity(演员)登记进去
//   3. 触发一次演示 Workflow,打印结果
// 【类比】你可以把 main.go 想象成"制片人":
//   他不演戏、不写剧本,但他租场地、雇剧团、喊 Action、然后等结果。
// -----------------------------------------------------------------------------

package main

import (
	"context"
	"log"

	"github.com/CXP-shawn/temporal-ai-agent-tutorial-go/activities"
	"github.com/CXP-shawn/temporal-ai-agent-tutorial-go/workflows"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// TaskQueue 是 Workflow/Activity 的"广播频道名"。
// Worker 监听这个队列,Client 也把任务扔进这个队列,二者靠它对接。
const TaskQueue = "ai-agent-queue"

func main() {
	// ---- 1. 连接 Temporal Server ----
	// localhost:7233 是 `temporal server start-dev` 启动时监听的 gRPC 端口。
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort, // 默认就是 127.0.0.1:7233
	})
	if err != nil {
		log.Fatalf("无法连接 Temporal Server(是不是忘了跑 temporal server start-dev?): %v", err)
	}
	defer c.Close() // 程序退出时关闭连接

	// ---- 2. 启动 Worker(剧团) ----
	// Worker 是一个长驻进程,负责从 Temporal Server 拉活儿、执行、上报结果。
	w := worker.New(c, TaskQueue, worker.Options{})

	// 把 Workflow(剧本)登记到剧团:告诉剧团"我会演这出戏"
	w.RegisterWorkflow(workflows.AgentWorkflow)

	// 把 Activity(演员)登记到剧团:告诉剧团"我会干这类脏活"
	w.RegisterActivity(activities.ExecuteToolActivity)

	// 在后台启动 Worker(非阻塞模式)
	// 注意:生产环境一般用 w.Run(...) 阻塞运行;这里为了同进程演示,用 Start + 手动触发。
	if err := w.Start(); err != nil {
		log.Fatalf("Worker 启动失败: %v", err)
	}
	defer w.Stop()

	// ---- 3. 触发一次演示 Workflow ----
	workflowOptions := client.StartWorkflowOptions{
		ID:        "demo-agent-run", // Workflow 实例 ID,同名只能同时跑一个
		TaskQueue: TaskQueue,
	}
	input := workflows.AgentInput{UserID: "demo-user"} // 传给剧本的开场参数

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.AgentWorkflow, input)
	if err != nil {
		log.Fatalf("触发 Workflow 失败: %v", err)
	}
	log.Printf("Workflow 已启动: WorkflowID=%s RunID=%s", we.GetID(), we.GetRunID())

	// ---- 4. 等 Workflow 跑完,拿最终结果 ----
	var result string
	if err := we.Get(context.Background(), &result); err != nil {
		log.Fatalf("Workflow 执行出错: %v", err)
	}
	log.Printf("最终结果: %s", result)
}
