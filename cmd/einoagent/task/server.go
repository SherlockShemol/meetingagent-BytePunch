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
	"encoding/json"
	"mime"
	"path/filepath"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"

	"meetingagent/redis" // 新增redis导入

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/task"
)

//go:embed static/*
var webContent embed.FS

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
			c.JSON(consts.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		// 新增任务生成功能
		if req.Action == "generate_from_summary" {
			meetingID := req.List.MeetingID
			if meetingID == "" {
				c.JSON(consts.StatusBadRequest, utils.H{"error": "meeting_id is required"})
				return
			}

			// 从Redis获取会议数据
			data, err := redis.Client.Get(ctx, "meeting:"+meetingID).Result()
			if err != nil {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"status": "error",
					"error":  "获取会议数据失败: " + err.Error(),
				})
				return
			}

			// 解析会议数据
			var meetingData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &meetingData); err != nil {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"status": "error",
					"error":  "解析会议数据失败: " + err.Error(),
				})
				return
			}

			// 调用agent生成任务
			summary, _ := meetingData["summary"].(string)
			generatedTasks, err := taskTool.GenerateTasksFromText(ctx, summary)
			if err != nil {
				c.JSON(consts.StatusInternalServerError, utils.H{
					"status": "error",
					"error":  "生成任务失败: " + err.Error(),
				})
				return
			}

			// 返回生成的任务列表
			c.JSON(consts.StatusOK, map[string]interface{}{
				"status":    "success",
				"task_list": generatedTasks,
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

func GenerateTasksFromSummary(ctx context.Context, summary string) ([]string, error) {
	// 调用agent生成任务
	generatedTasks, err := taskTool.GenerateTasksFromText(ctx, summary)
	if err != nil {
		return nil, err
	}
	return generatedTasks, nil
}
