package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"meetingagent/handlers/agent"
	"meetingagent/models"
	"meetingagent/pkg/env"
	"meetingagent/redis"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sse"
	"github.com/joho/godotenv"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

const prompt string = `
# Role: 会议总结助手

## Core Competencies
- 你获取的会议数据只会是文本格式，不会有其他非文本的数据输入
- 识别并提取会议内容中的关键讨论点、决策和行动项
- 将非结构化的会议数据（记录、笔记）转化为简洁、结构化的摘要
- 高精度分析和综合自然语言内容
- 理解常见的会议工作流程，并与协作工具（例如日历、任务跟踪器）集成

## Interaction Guidelines
- Before responding, ensure you:
  • 全面分析所有提供的会议材料（记录、笔记或音频），以捕捉关键主题、决策和任务
  • 如果任何背景信息不明确或不完整（例如缺少参与者、术语模糊），向用户询问以澄清
  • 根据用户偏好或内容复杂性调整摘要格式（例如项目符号、段落或表格）

- When providing assistance:
  • 提供简洁、条理清晰的摘要，优先考虑清晰度和可读性
  • 突出关键元素：做出的决策、分配的行动项（包括责任人和截止日期）以及主要讨论点
  • 引用会议中的简短相关示例或话语，以支持关键结论
  • 提供可操作的后续步骤，例如基于这次会议提出的任务进行记录


- If a request exceeds your capabilities:
  • 如果无法处理某些输入（例如未转录的音频），明确说明限制并建议解决方案（例如“仅支持文本文件输入”）

- If the question is compound or complex:
  • 将冗长或多方面的讨论分解为清晰的类别（例如主题、决策、任务）
  • 通过交叉引用所有提供材料，确保不遗漏关键细节
  • 一步步思考，保证答案的正确性和完整性

## Context Information
- 当前日期: {date}
- 会议材料: |-
==== meeting_doc start ====
  {meeting_transcript_or_notes}
==== meeting_doc end ====
`

// CreateMeeting handles the creation of a new meeting
func CreateMeeting(ctx context.Context, c *app.RequestContext) {
	log.Println("CreateMeeting 被调用")
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

	// 使用Redis
	err = redis.Client.Set(ctx, "meeting:"+response.ID, jsonBody, 0).Err()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存到Redis失败"})
		return
	}
	//log.Printf("create meeting: %s\n", response.ID)
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
	log.Println("GetMeetingSummary 被调用")
	meetingID := c.Query("meeting_id")
	if meetingID == "" {
		c.JSON(consts.StatusBadRequest, utils.H{"error": "meeting_id is required"})
		return
	}

	fmt.Printf("meetingID: %s\n", meetingID)
	// TODO: Implement actual summary retrieval logic
	summary, err := LLM()

	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"content": summary,
	}

	c.JSON(consts.StatusOK, response)
}

// LLM
type MeetingContent struct {
	Contents []struct {
		TimeFrom string `json:"time_from"`
		TimeTo   string `json:"time_to"`
		User     string `json:"user"`
		Content  struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"contents"`
}

func LLM() (string, error) {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v\n", err)
		return "", err
	}
	// 读取content.json
	data, err := os.ReadFile("d:/Github/meetingagent-BytePunch/example/content.json")
	if err != nil {
		log.Printf("读取content.json失败: %v\n", err)
		return "", err
	}
	var meeting MeetingContent
	err = json.Unmarshal(data, &meeting)
	if err != nil {
		log.Printf("解析content.json失败: %v\n", err)
		return "", err
	}

	// 拼接所有文本内容
	var allText []string
	for _, item := range meeting.Contents {
		allText = append(allText, item.Content.Text)
	}
	inputText := strings.Join(allText, "\n")

	client := arkruntime.NewClientWithApiKey(
		os.Getenv("ARK_API_KEY"),
		arkruntime.WithBaseUrl("https://ark.cn-beijing.volces.com/api/v3"),
	)
	ctx := context.Background()
	req := model.CreateChatCompletionRequest{
		Model: os.Getenv("ARK_CHAT_MODEL"),
		Messages: []*model.ChatCompletionMessage{
			{
				Role: model.ChatMessageRoleSystem,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(prompt),
				},
			},
			{
				Role: model.ChatMessageRoleUser,
				Content: &model.ChatCompletionMessageContent{
					StringValue: volcengine.String(inputText),
				},
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Printf("standard chat error: %v", err)
		return "", err
	}
	return *resp.Choices[0].Message.Content.StringValue, nil
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
