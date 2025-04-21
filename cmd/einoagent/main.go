package main

import (
	"context"
	"log"
	"time"

	"meetingagent/handlers"
	"meetingagent/pkg/env"
	"meetingagent/redis"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/agent"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/task"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func init() {
	// check some essential envs
	env.MustHasEnvs("ARK_CHAT_MODEL", "ARK_EMBEDDING_MODEL", "ARK_API_KEY")
}

func main() {

	redis.Init()
	h := server.Default()
	h.Use(Logger())

	taskGroup := h.Group("/task")
	if err := task.BindRoutes(taskGroup); err != nil {
		log.Fatal("failed to bind task routes:", err)
	}

	// 注册 agent 路由组
	handlersGroup := h.Group("/handlers")
	if err := agent.BindRoutes(handlersGroup); err != nil {
		log.Fatal("failed to bind agent routes:", err)
	}

	// Redirect root path to /agent
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Redirect(302, []byte("/agent"))
	})

	// Register API routes first
	h.POST("/meeting", handlers.CreateMeeting)
	h.GET("/meeting", handlers.ListMeetings)
	h.GET("/summary", handlers.GetMeetingSummary)
	h.GET("/chat", handlers.HandleChat)

	// Serve static files
	h.StaticFS("/", &app.FS{
		Root:               "./cmd/einoagent/task/static",
		PathRewrite:        app.NewPathSlashesStripper(1),
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: true,
	})

	// Start server
	h.Spin()
}

// Logger 记录 HTTP 请求日志
func Logger() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := string(c.Request.URI().Path())
		method := string(c.Request.Method())

		// 处理请求
		c.Next(ctx)

		// 记录请求信息
		latency := time.Since(start)
		statusCode := c.Response.StatusCode()
		log.Printf("[HTTP] %s %s %d %v\n", method, path, statusCode, latency)

	}
}
