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

package redis

import (
	"context"
	"fmt"
	"os" // 新增导入 os 包
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

const (
	RedisPrefix = "eino:doc:"
	IndexName   = "vector_index"

	ContentField  = "content"
	MetadataField = "metadata"
	VectorField   = "content_vector"
	DistanceField = "distance"
)

var initOnce sync.Once

func Init() error {
	var err error
	initOnce.Do(func() {
		// 从环境变量读取 Redis 地址，如果未设置则使用默认值
		redisAddr := os.Getenv("REDIS_ADDR")
		if redisAddr == "" {
			redisAddr = "localhost:6379" // 保留默认值
			fmt.Println("[Redis Init] REDIS_ADDR not set, using default:", redisAddr)
		} else {
			fmt.Println("[Redis Init] Using REDIS_ADDR from env:", redisAddr)
		}

		// TODO: 从配置或环境变量读取 Dimension
		// 这里暂时保持硬编码，但最好也改为配置化
		dimension := 4096
		fmt.Println("[Redis Init] Using Dimension:", dimension)

		err = InitRedisIndex(context.Background(), &Config{
			RedisAddr: redisAddr, // 使用从环境变量读取或默认的地址
			Dimension: dimension, // 使用获取到的维度
		})
		if err != nil {
			// 明确打印初始化错误
			fmt.Printf("[Redis Init] Error initializing Redis index: %v\n", err)
		} else {
			fmt.Println("[Redis Init] Redis index initialization check completed.")
		}
	})
	return err // 返回初始化过程中遇到的错误
}

type Config struct {
	RedisAddr string
	Dimension int
}

func InitRedisIndex(ctx context.Context, config *Config) (err error) {
	if config.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}
	fmt.Printf("[Redis InitRedisIndex] Attempting to connect to Redis at %s\n", config.RedisAddr)

	client := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Protocol: 2,
	})

	// 修改 defer，确保无论如何都关闭客户端
	defer func() {
		closeErr := client.Close()
		if closeErr != nil {
			fmt.Printf("[Redis InitRedisIndex] Error closing Redis client: %v\n", closeErr)
		}
		// 如果函数本身返回错误，保留原始错误
		// 如果关闭时出错，但函数本身没出错，可以考虑是否覆盖 err
	}()

	if err = client.Ping(ctx).Err(); err != nil {
		// 提供更清晰的连接失败信息
		return fmt.Errorf("failed to connect to Redis at %s: %w", config.RedisAddr, err)
	}
	fmt.Printf("[Redis InitRedisIndex] Successfully connected to Redis at %s\n", config.RedisAddr)

	indexName := fmt.Sprintf("%s%s", RedisPrefix, IndexName)
	fmt.Printf("[Redis InitRedisIndex] Checking for index: %s\n", indexName)

	// 检查是否存在索引
	// 使用 FT._LIST 检查更可靠
	indices, err := client.Do(ctx, "FT._LIST").StringSlice()
	if err != nil {
		// 如果 FT._LIST 命令本身失败，可能是 RediSearch 模块未加载
		if strings.Contains(err.Error(), "unknown command") {
			return fmt.Errorf("redis command FT._LIST failed, ensure RediSearch module is loaded on Redis server at %s: %w", config.RedisAddr, err)
		}
		return fmt.Errorf("failed to list existing Redis Search indices: %w", err)
	}

	indexExists := false
	for _, existingIndex := range indices {
		if existingIndex == indexName {
			indexExists = true
			break
		}
	}

	if indexExists {
		fmt.Printf("[Redis InitRedisIndex] Index '%s' already exists.\n", indexName)
		return nil // 索引已存在，无需创建
	}

	fmt.Printf("[Redis InitRedisIndex] Index '%s' not found, attempting to create...\n", indexName)
	// Create new index
	createIndexArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", RedisPrefix,
		"SCHEMA",
		ContentField, "TEXT",
		MetadataField, "TEXT",
		VectorField, "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return fmt.Errorf("failed to create index '%s': %w", indexName, err)
	}
	fmt.Printf("[Redis InitRedisIndex] Successfully created index '%s'.\n", indexName)

	// 验证索引是否创建成功 (可选，因为 Do 没有报错通常就表示成功了)
	// if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
	// 	return fmt.Errorf("failed to verify index creation '%s': %w", indexName, err)
	// }
	// fmt.Printf("[Redis InitRedisIndex] Verified index creation '%s'.\n", indexName)

	return nil
}
