package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"meetingagent/handlers/agent"
	"meetingagent/models"
	"meetingagent/pkg/env"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sse"
)

// CreateMeeting handles the creation of a new meeting
func CreateMeeting(ctx context.Context, c *app.RequestContext) {
	var reqBody map[string]interface{}
	if err := c.BindJSON(&reqBody); err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		c.JSON(consts.StatusBadRequest, utils.H{"error": err.Error()})
		return
	}

	fmt.Printf("create meeting: %s\n", string(jsonBody))

	// TODO: Implement actual meeting creation logic
	response := models.PostMeetingResponse{
		ID: "meeting_" + time.Now().Format("20060102150405"),
	}

	c.JSON(consts.StatusOK, response)
}

// ListMeetings handles listing all meetings
func ListMeetings(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement actual meeting retrieval logic
	response := models.GetMeetingsResponse{
		Meetings: []models.Meeting{
			{
				ID: "meeting_123",
				Content: map[string]interface{}{
					"title":        "Sample Meeting",
					"description":  "This is a sample meeting",
					"participants": []string{"John Doe", "Jane Smith"},
					"start_time":   "2025-04-20 08:00:00",
					"end_time":     "2025-04-20 09:00:00",
					"content":      "This is the content of the meeting",
				},
			},
		},
	}

	c.JSON(consts.StatusOK, response)
}

// GetMeetingSummary handles retrieving a meeting summary
func GetMeetingSummary(ctx context.Context, c *app.RequestContext) {
	meetingID := c.Query("meeting_id")
	if meetingID == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "meeting_id is required"})
		return
	}
	fmt.Printf("meetingID: %s\n", meetingID)

	// TODO: Implement actual summary retrieval logic
	response := map[string]interface{}{
		"content": `
		Meeting summary for ` + meetingID + `## Summary
we talked about the project and the next steps, we will have a call next week to discuss the project in more detail.

......
		`,
	}

	c.JSON(consts.StatusOK, response)
}

// HandleChat handles the SSE chat session
func HandleChat(ctx context.Context, c *app.RequestContext) {
	meetingID := c.Query("meeting_id")
	sessionID := c.Query("session_id")
	message := c.Query("message")

	// check some essential envs
	env.MustHasEnvs("ARK_CHAT_MODEL", "ARK_EMBEDDING_MODEL", "ARK_API_KEY")

	if meetingID == "" || sessionID == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "meeting_id and session_id are required"})
		return
	}

	if message == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "message is required"})
		return
	}

	fmt.Printf("meetingID: %s, sessionID: %s, message: %s\n", meetingID, sessionID, message)

	// Set SSE headers
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")

	sr, err := agent.RunAgent(ctx, sessionID, message)
	if err != nil {
		log.Printf("[Chat] Error running agent: %v\n", err)
		c.JSON(consts.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Create SSE stream
	s := sse.NewStream(c)
	defer func() {
		sr.Close()
		c.Flush()

		log.Printf("[Chat] Finished chat with sessionID: %s\n", sessionID) // Use sessionID for logging consistency
	}()

outer:
	for {
		select {
		case <-ctx.Done():
			log.Printf("[Chat] Context done for chat sessionID: %s\n", sessionID) // Use sessionID for logging consistency
			return
		default:
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				log.Printf("[Chat] EOF received for chat sessionID: %s\n", sessionID) // Use sessionID for logging consistency
				break outer
			}
			if err != nil {
				log.Printf("[Chat] Error receiving message: %v\n", err)
				break outer
			}

			// 构造符合API文档格式的响应数据
			responseData := map[string]interface{}{
				"data": map[string]interface{}{
					"message":   msg.Content,
					"timestamp": time.Now().UTC().Format(time.RFC3339), // 使用当前UTC时间
					"sender":    "Agent",                               // 发送者标识为Agent
				},
			}

			// 将响应数据序列化为JSON
			jsonData, err := json.Marshal(responseData)
			if err != nil {
				log.Printf("[Chat] Error marshalling JSON: %v\n", err)
				continue // 如果序列化失败，跳过这条消息
			}

			// 发布JSON数据
			err = s.Publish(&sse.Event{
				Data: jsonData,
			})
			if err != nil {
				log.Printf("[Chat] Error publishing message: %v\n", err)
				break outer
			}
		}
	}
}
