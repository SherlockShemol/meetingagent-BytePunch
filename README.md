# 会议总结助手

## 简介
这个项目是一个会议总结助手，旨在帮助用户更好地组织和记录会议内容。用户通过输入会议文本(仅支持JSON)，助手可以自动识别会议内容并生成结构化的会议摘要。

## 功能特性
- 功能1：存储会议信息
- 功能2：根据会议信息生成会议摘要，并可通过redis查询
- 功能3：可与大模型交互，通过KnowledgeIndexing检索增强，对会议内容有更精准的回答。
- 功能4：可根据用户的需求，根据会议摘要生成任务清单。

## 安装
```bash
# 克隆仓库
git clone https://github.com/SherlockShemol/meetingagent-BytePunch.git

# 在项目目录下，安装依赖
go mod tidy  # 或其他包管理器命令
```

## 配置
1.配置文件设置:
  ```bash
  cp config.yaml.example config.yaml
  ```

2.编辑 config.yaml 文件，填入您的 API Key 和其他配置:
```yaml
# 必填，
# 火山云方舟 ChatModel 的 Endpoint ID
ARK_CHAT_MODEL="doubao-1-5-thinking-pro-250415"
# 火山云方舟 向量化模型的 Endpoint ID
ARK_EMBEDDING_MODEL="doubao-embedding-large-text-240915"
# 火山云方舟的 API Key
ARK_API_KEY="your_ark_api_key"
# Redis Server 的地址，不填写时，默认是 localhost:6379
export REDIS_ADDR=
```

## 项目启动
```bash
# 在项目目录下，通过docker-compose.yml启动redis-stack
docker-compose up -d  # 或其他启动命令

cd cmd/einoagent

go run main.go

之后通过浏览器访问localhost:8080即可

```

## 项目结构
- `cmd/einoagent`: 项目的主要业务逻辑。
  - `main.go`: 项目的入口文件。
  - `agent/agent.go`: 包含项目的辅助函数和工具。
  - `data/`: Agent的memory存储位置。。
  - `meetings/`: 处理json格式后切分成的.md文件存储位置
  - `task/`: 前端的生成
     - `static/`: 前端的静态文件。 
- `einoagent/`: Eino构建起的Agent框架
- `examples/`: 样例的输入文件。
- `handlers/`: 项目主要的后端逻辑，处理文本输入，摘要查询，对话生成以及任务生成。
- `model/meeting.go`: 一些会议的结构体。
- `pkg/`: Eino框架中向量化模型的连接和操作。
- `knowledgeindexing/`: 文件夹下包含knowledge indexing的相关文件。
- `rag/knowledgeindexing.go`: Eino框架中向量化模型的连接和操作。
- `redis/redis.go`: redis的连接和操作