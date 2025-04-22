应用场景即会议后根据会议json文件（example/content.json）生成会议纪要，并根据会议纪要内容生成任务/ToDo清单，用户也可根据自己的需要通过对话的方式在清单中添加或删除任务。

运行项目
```go
// 首先打开docker容器
docker-compose up -d
// 在.env文件中配置自己的key
go run cmd/einoagent/main.go
```

