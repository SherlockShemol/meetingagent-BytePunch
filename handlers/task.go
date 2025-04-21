package handlers

import (
	"context"
	"embed"
	"meetingagent/pkg/tool/task"
	"mime"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

var webContent embed.FS

func GetTaskList(ctx context.Context, c *app.RequestContext) {

	taskTool, _ := task.NewTaskToolImpl(ctx, &task.TaskToolConfig{
		Storage: task.GetDefaultStorage(),
	})

	var req task.TaskRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	resp, err := taskTool.Invoke(ctx, &req)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(consts.StatusOK, resp)

	// 静态文件服务
	content, err := webContent.ReadFile("index.html")
	if err != nil {
		c.String(consts.StatusNotFound, "File not found")
		return
	}
	c.Header("Content-Type", "text/html")
	c.Write(content)

	file := c.Param("file")
	content, _ = webContent.ReadFile(file)

	contentType := mime.TypeByExtension(filepath.Ext(file))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Write(content)

}
