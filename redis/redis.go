package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func Init() {
	Client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis 地址
		Password: "",               // 密码
		DB:       0,                // 默认DB
	})

	// 测试连接
	_, err := Client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("无法连接Redis:", err)
	}
	log.Println("Redis连接成功")
}
