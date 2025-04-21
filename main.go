package main

import (
	"context"
	"log"
	"time"

	"meetingagent/handlers"
	"meetingagent/redis"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func main() {

	redis.Init()
	h := server.Default()
	h.Use(Logger())

	// Register API routes first
	h.POST("/meeting", handlers.CreateMeeting)
	h.GET("/meeting", handlers.ListMeetings)
	h.GET("/summary", handlers.GetMeetingSummary)
	h.GET("/chat", handlers.HandleChat)
	h.GET("/task", handlers.GetTaskList)

	// Serve static files
	h.StaticFS("/", &app.FS{
		Root:               "./static",
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
