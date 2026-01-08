# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

ChatRecommend 是一个用 Go 语言实现的智能聊天补全系统，支持接入微信、抖音等聊天应用，提供类似 Cursor 的自动补全功能。系统通过 Python 脚本调用大模型 API（OpenAI、Anthropic 等）来实现智能补全。

## 常用命令

### 构建和运行

```bash
# 下载 Go 依赖
go mod download

# 下载 Python 依赖
pip install -r python/requirements.txt

# 运行服务器
go run cmd/server/main.go

# 构建（如果需要）
go build -o bin/server cmd/server/main.go
```

### 配置

项目使用 `config.yaml` 作为配置文件。首次运行前需要：
1. 复制并编辑配置文件，设置大模型 API 密钥
2. 确保配置文件中的 `llm.api.api_key` 已正确设置

### 数据库

项目使用 SQLite 数据库，默认路径为 `./data/chat.db`。数据库会在首次运行时自动创建并迁移表结构。

## 架构说明

### 核心组件

1. **上下文管理器 (`internal/context/`)**
   - 负责构建对话上下文，整合摘要、风格和近期消息
   - 智能截断上下文以适应 token 限制
   - 优先保留重要信息（摘要 > 风格 > 近期消息）

2. **摘要管理器 (`internal/summary/`)**
   - 自动生成和维护对话摘要
   - 基于消息数量和时间阈值自动更新
   - 通过 LLM 接口调用 Python 脚本生成摘要

3. **风格管理器 (`internal/style/`)**
   - 学习用户的语言风格特征（词汇、句长、语气等）
   - 每个用户的风格独立存储
   - 在生成补全时应用学习到的风格

4. **自动补全引擎 (`internal/autocomplete/`)**
   - 核心业务逻辑，协调上下文、摘要、风格管理器
   - 实现请求去抖（debounce）功能
   - 调用大模型生成补全建议

5. **API 层 (`internal/api/`)**
   - HTTP 接口：`/api/chat/complete`、`/api/chat/message`、`/api/chat/history/:id`
   - WebSocket 接口：`/ws` 用于实时补全
   - 异步更新摘要和风格，不阻塞主流程

### 数据流

1. **保存消息流程**：
   - API 接收消息 → 保存到数据库 → 异步更新摘要和风格

2. **生成补全流程**：
   - API 接收请求 → 构建上下文（摘要+风格+近期消息）→ 调用大模型 → 返回建议

3. **摘要更新流程**：
   - 检查阈值（消息数量/时间）→ 调用 Python 脚本 → 保存摘要到数据库

### 关键设计

- **两层架构**：Go 后端 + Python 大模型客户端
- **异步更新**：摘要和风格更新在 goroutine 中异步执行
- **请求去抖**：通过 `debounceMap` 实现请求合并，减少大模型调用
- **智能截断**：上下文过长时优先保留摘要和风格，截断历史消息

## 重要配置

### 摘要配置 (`config.yaml`)

```yaml
summary:
  update_threshold_messages: 100  # 达到此消息数后触发更新
  update_threshold_hours: 24      # 达到此时间后触发更新
  max_summary_tokens: 500         # 摘要最大长度
  auto_update: true               # 是否启用自动更新
```

### 风格配置

```yaml
style:
  learning_messages_count: 50     # 用于风格学习的消息数量
  update_threshold_messages: 20   # 风格更新阈值
  enabled: true                   # 是否启用风格学习
```

### 上下文配置

```yaml
context:
  max_context_tokens: 4000        # 最大上下文长度
  recent_messages_count: 50       # 近期消息数量
  history_retention_count: 1000   # 保留的历史消息数量
```

## 开发注意事项

### 添加新功能时

1. 如果需要修改数据库结构，更新 `internal/models/models.go` 中的模型定义
2. 新增配置项需要在 `internal/config/config.go` 中添加结构体字段
3. 修改 API 接口时同时更新 HTTP 和 WebSocket 处理器

### 大模型相关

- 大模型调用通过 Python 脚本 (`python/llm_client.py`) 实现
- 支持 OpenAI 和 Anthropic API
- 如需支持新的大模型，修改 Python 脚本和 Go 中的 `llm.Client`

### 性能优化

- 请求去抖时间默认 300ms，可在配置中调整
- 异步操作使用 goroutine，注意错误处理
- 上下文截断使用粗略估算（1 token ≈ 3 字符），可根据需要改进

## Python 依赖

项目需要以下 Python 包：
- `openai>=1.0.0`
- `anthropic>=0.7.0`

安装命令：`pip install -r python/requirements.txt`

## 启动检查清单

1. Go 依赖已安装：`go mod download`
2. Python 依赖已安装：`pip install -r python/requirements.txt`
3. 配置文件已设置 API 密钥：编辑 `config.yaml`
4. 数据目录存在：`data/` 目录会自动创建
5. 日志目录存在：`logs/` 目录会自动创建（如果配置为文件输出）


## 用户交互要求

我提出需求之后，你不应该马上写代码，而是应该做这些事情：

1. 复述我的需求
2. 设计集成测试用例
3. 描述你的方案，和这个方案有哪些问题

**修改现有代码前需要找我确认**