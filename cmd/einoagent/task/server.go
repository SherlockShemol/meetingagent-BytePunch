/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package task

import (
	"context"
	"embed"
	"fmt"
	"mime"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	// 新增redis导入

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/task"
)

//go:embed static/*
var webContent embed.FS

// 参数结构体
type TodoUpdateParams struct {
	ID        string  `json:"id" jsonschema:"description=id of the todo"`
	Content   *string `json:"content,omitempty" jsonschema:"description=content of the todo"`
	StartedAt *int64  `json:"started_at,omitempty" jsonschema:"description=start time in unix timestamp"`
	Deadline  *int64  `json:"deadline,omitempty" jsonschema:"description=deadline of the todo in unix timestamp"`
	Done      *bool   `json:"done,omitempty" jsonschema:"description=done status"`
}

// BindRoutes 注册路由
func BindRoutes(r *route.RouterGroup) error {
	ctx := context.Background()

	taskTool, err := task.NewTaskToolImpl(ctx, &task.TaskToolConfig{
		Storage: task.GetDefaultStorage(),
	})
	if err != nil {
		return err
	}

	// API 处理

	r.POST("/api", func(ctx context.Context, c *app.RequestContext) {
		var req task.TaskRequest
		if err := c.Bind(&req); err != nil {
			c.JSON(consts.StatusBadRequest, map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		resp, err := taskTool.Invoke(ctx, &req)
		if err != nil {
			// 记录详细错误信息
			fmt.Printf("Task API error: %v\n", err)
			c.JSON(consts.StatusInternalServerError, map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}
		c.JSON(consts.StatusOK, resp)
	})

	// 静态文件服务
	r.GET("/", func(ctx context.Context, c *app.RequestContext) {
		content, err := webContent.ReadFile("static/index.html")
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}
		c.Header("Content-Type", "text/html")
		c.Write(content)
	})

	r.GET("/:file", func(ctx context.Context, c *app.RequestContext) {
		file := c.Param("file")
		content, err := webContent.ReadFile("static/" + file)
		if err != nil {
			c.String(consts.StatusNotFound, "File not found")
			return
		}

		contentType := mime.TypeByExtension(filepath.Ext(file))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Header("Content-Type", contentType)
		c.Write(content)
	})

	return nil
}
