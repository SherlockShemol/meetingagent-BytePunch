package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"meetingagent/cmd/einoagent/agent"
	"meetingagent/models"
	"meetingagent/pkg/env"
	"meetingagent/rag"
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
# Role: 会议总结和任务/Todo生成助手

## Core Competencies
- 你获取的会议数据只会是文本格式，不会有其他非文本的数据输入
- 识别并提取会议内容中的关键讨论点、决策和行动项
- 将非结构化的会议数据（记录、笔记）转化为简洁、结构化的摘要
- 高精度分析和综合自然语言内容
- 生成的内容的标题应该简洁明了，且为一级标题
- 理解常见的会议工作流程，并与协作工具（例如日历、任务跟踪器）集成

## Interaction Guidelines
- Before responding, ensure you:
  • 全面分析所有提供的会议材料（记录、笔记或音频），以捕捉关键主题、决策和任务
  • 如果任何背景信息不明确或不完整（例如缺少参与者、术语模糊），向用户询问以澄清
  • 根据用户偏好或内容复杂性调整摘要格式（例如项目符号、段落或表格）

- When providing assistance:
  • 提供简洁、条理清晰的摘要，优先考虑清晰度和可读性
  • 突出会议主要的讨论点以及结果
  • 突出关键元素：做出的决策、分配的行动项（包括责任人和截止日期），决策和分配的行动项最好能按表格展示
  • 引用会议中的简短相关示例或话语，以支持关键结论
  • 如果生成的内容能够细分为多点，请将其分解为多个要点
  • 提供可操作的后续步骤，例如基于这次会议提出的任务进行记录


- If a request exceeds your capabilities:
  • 如果无法处理某些输入（例如未转录的音频），明确说明限制并建议解决方案（例如“仅支持文本文件输入”）

- If the question is compound or complex:
  • 将冗长或多方面的讨论分解为清晰的类别（例如主题、决策、任务）
  • 通过交叉引用所有提供材料，确保不遗漏关键细节
  • 一步步思考，保证答案的正确性和完整性

## Context Information
- 当前日期: {meetingDate}
- 会议材料: |-
==== meeting_doc start ====
  {meetingTranscript}
==== meeting_doc end ====
`

func init() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading.env file: %v\n", err)
	}
}

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

	//fmt.Printf("create meeting: %s\n", string(jsonBody))

	timestamp := time.Now().Format("20060102_150405")
	markdownFilePath := filepath.Join("meetings", fmt.Sprintf("%s.md", timestamp))
	if err := os.MkdirAll("meetings", 0755); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "Failed to create meetings directory: " + err.Error()})
		return
	}
	// 调用 convertJSONToMarkdown 生成 Markdown 文件
	if err := rag.ConvertJSONToMarkdown(jsonBody, markdownFilePath); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "Failed to convert JSON to Markdown: " + err.Error()})
		return
	}

	rag.IndexMarkdownFiles(ctx, "meetings")
	// TODO: Implement actual meeting creation logic
	response := models.PostMeetingResponse{
		ID: "meeting_" + time.Now().Format("20060102150405"),
	}

	meetingDate := time.Now().Format("2006-01-02")
	meetingTranscript := string(jsonBody)
	prompt := fmt.Sprintf(prompt, meetingDate, meetingTranscript)
	// 调用LLM生成总结
	summary, err := LLM(prompt)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "生成会议总结失败"})
		return
	}

	// 构建完整数据
	meetingData := models.Meeting{
		ID:        response.ID,
		Content:   reqBody,
		Summary:   summary,
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	completeData, err := json.Marshal(meetingData)
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "数据序列化失败"})
		return
	}
	//log.Printf("create meeting: %s\n", response.ID)

	err = redis.Client.Set(ctx, "meeting:"+response.ID, completeData, 0).Err()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "保存到Redis失败"})
		return
	}
	c.JSON(consts.StatusOK, response)
}

// ListMeetings handles listing all meetings
func ListMeetings(ctx context.Context, c *app.RequestContext) {
	// TODO: Implement actual meeting retrieval logic
	keys, err := redis.Client.Keys(ctx, "meeting:*").Result()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "无法获取会议列表"})
		return
	}

	var meetings []models.Meeting

	// 遍历每个键并获取会议数据
	for _, key := range keys {
		data, err := redis.Client.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Error retrieving meeting data for key %s: %v", key, err)
			continue
		}

		var meeting models.Meeting
		if err := json.Unmarshal([]byte(data), &meeting); err != nil {
			log.Printf("Error unmarshalling meeting data for key %s: %v", key, err)
			continue
		}

		meetings = append(meetings, meeting)
	}

	// 返回会议列表
	response := models.GetMeetingsResponse{
		Meetings: meetings,
	}
	//log.Printf("response:= %+v", response)

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

	// 从Redis获取会议数据
	data, err := redis.Client.Get(ctx, "meeting:"+meetingID).Result()
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "获取会议数据失败"})
		return
	}

	// 解析JSON数据
	var meetingData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &meetingData); err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": "解析会议数据失败"})
		return
	}
	if err != nil {
		c.JSON(consts.StatusInternalServerError, utils.H{"error": err.Error()})
		return
	}

	response := map[string]interface{}{
		"meeting_id": meetingID,
		"summary":    meetingData["summary"],
		"created_at": meetingData["created_at"],
	}
	log.Printf("读取redis的会议内容")
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

func LLM(prompt string) (string, error) {
	// 读取content.json
	inputText := prompt
	//data := reqBody
	// var meeting MeetingContent
	// err := json.Unmarshal(data, &meeting)
	// if err != nil {
	// 	log.Printf("解析content.json失败: %v\n", err)
	// 	return "", err
	// }

	// 拼接所有文本内容
	// var allText []string
	// for _, item := range meeting.Contents {
	// 	allText = append(allText, item.Content.Text)
	// }
	// inputText := strings.Join(allText, "\n")

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
					StringValue: volcengine.String(inputText),
				},
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	log.Printf("resp生成完毕")
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
