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

package einoagent

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var systemPrompt = `
# Role: Eino Meeting Assistant

## Core Competencies
- 你获取的会议数据只会是文本格式，不会有其他非文本的数据输入
- 识别并提取会议内容中的关键讨论点、决策和行动项
- 将非结构化的会议数据（记录、笔记）转化为简洁、结构化的摘要
- 高精度分析和综合自然语言内容
- 理解常见的会议工作流程，并与协作工具（例如日历、任务跟踪器）集成
- 可以根据会议内容以及用户的要求生成任务，并加入到任务清单中

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

- 如果用户要求将某事物加入到任务中，需要调用task工具，将其加入到任务中，并返回任务ID。
`

type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

// newChatTemplate component initialization function of node 'ChatTemplate' in graph 'EinoAgent'
func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	// TODO Modify component configuration here.
	config := &ChatTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}
