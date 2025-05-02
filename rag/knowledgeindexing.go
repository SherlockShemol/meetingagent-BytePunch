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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/env"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/redis/go-redis/v9"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/eino/knowledgeindexing"
)

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

func init() {
	// check some essential envs
	env.MustHasEnvs("ARK_API_KEY", "ARK_EMBEDDING_MODEL")
}

// SplitMarkdownFile 将 Markdown 文件切分为指定数量的文件
func SplitMarkdownFile(inputPath string, maxChunks int) ([]string, error) {
	// 读取原始文件内容
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("read file %s failed: %w", inputPath, err)
	}

	// 如果内容为空或不需要切分，直接返回原文件路径
	if len(content) == 0 || maxChunks <= 1 {
		return []string{inputPath}, nil
	}

	// 按行分割内容
	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)
	linesPerChunk := (totalLines + maxChunks - 1) / maxChunks // 向上取整

	var outputFiles []string
	currentChunk := strings.Builder{}
	chunkCount := 0
	lineCount := 0

	for i, line := range lines {
		currentChunk.WriteString(line + "\n")
		lineCount++

		// 当达到每块的行数或到达最后一行时，保存当前块
		if lineCount >= linesPerChunk || i == totalLines-1 {
			// 生成新的文件路径
			chunkFileName := fmt.Sprintf("%s.part%d.md", strings.TrimSuffix(inputPath, ".md"), chunkCount)
			err := os.WriteFile(chunkFileName, []byte(currentChunk.String()), 0644)
			if err != nil {
				return nil, fmt.Errorf("write chunk file %s failed: %w", chunkFileName, err)
			}
			outputFiles = append(outputFiles, chunkFileName)
			currentChunk.Reset()
			chunkCount++
			lineCount = 0
		}
	}

	return outputFiles, nil
}

func main() {
	ctx := context.Background()

	// 文件路径
	err := IndexMarkdownFiles(ctx, "./example/")
	if err != nil {
		panic(err)
	}

	fmt.Println("index success")
}

func ConvertJSONToMarkdown(jsondata []byte, markdownFilePath string) error {

	var meetingContent MeetingContent
	if err := json.Unmarshal(jsondata, &meetingContent); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 创建 Markdown 文件
	markdownFile, err := os.Create(markdownFilePath)
	if err != nil {
		return fmt.Errorf("failed to create Markdown file: %w", err)
	}
	defer markdownFile.Close()
	log.Printf("create markdown file: %s\n", markdownFilePath)
	// 转换内容并写入 Markdown 文件
	for _, content := range meetingContent.Contents {
		line := fmt.Sprintf("%s-%s %s: %s\n", content.TimeFrom, content.TimeTo, content.User, content.Content.Text)
		if _, err := markdownFile.WriteString(line); err != nil {
			return fmt.Errorf("failed to write to Markdown file: %w", err)
		}
	}

	return nil
}

func IndexMarkdownFiles(ctx context.Context, dir string) error {
	runner, err := knowledgeindexing.BuildKnowledgeIndexing(ctx)
	if err != nil {
		return fmt.Errorf("build index graph failed: %w", err)
	}

	// 遍历 dir 下的所有 markdown 文件
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir failed: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			fmt.Printf("[skip] not a md file: %s\n", path)
			return nil
		}

		fmt.Printf("[start] indexing file: %s\n", path)

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s failed: %w", path, err)
		}

		// 如果文件内容超过 maxChunkSize，切分为多个文件
		var filesToIndex []string
		if len([]rune(string(content))) > 4096 {
			fmt.Printf("[split] file %s exceeds max size, splitting into %d parts\n", path, 10)
			filesToIndex, err = SplitMarkdownFile(path, 5)
			if err != nil {
				return fmt.Errorf("split file %s failed: %w", path, err)
			}
		} else {
			filesToIndex = []string{path}
		}

		// 调用 runner 进行索引
		for _, filePath := range filesToIndex {
			fmt.Printf("[start] indexing file: %s\n", filePath)
			ids, err := runner.Invoke(ctx, document.Source{URI: filePath})
			if err != nil {
				return fmt.Errorf("invoke index graph for file %s failed: %w", filePath, err)
			}
			fmt.Printf("[done] indexing file: %s, len of parts: %d\n", filePath, len(ids))
		}
		log.Printf("index完成")
		return nil
	})

	return err
}

type RedisVectorStoreConfig struct {
	RedisKeyPrefix string
	IndexName      string
	Embedding      embedding.Embedder
	Dimension      int
	MinScore       float64
	RedisAddr      string
}

func InitVectorIndex(ctx context.Context, config *RedisVectorStoreConfig) (err error) {
	if config.Embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}
	if config.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}

	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})

	// 确保在错误时关闭连接
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if err = client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	indexName := fmt.Sprintf("%s%s", config.RedisKeyPrefix, config.IndexName)

	// 检查是否存在索引
	exists, err := client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "Unknown index name") {
			return fmt.Errorf("failed to check if index exists: %w", err)
		}
		err = nil
	} else if exists != nil {
		return nil
	}

	// Create new index
	createIndexArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", config.RedisKeyPrefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 验证索引是否创建成功
	if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		return fmt.Errorf("failed to verify index creation: %w", err)
	}

	return nil
}
