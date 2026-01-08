# ChatRecommend - 智能聊天补全系统

一个用Go语言实现的工程化聊天补全系统，支持接入微信、抖音等聊天应用，提供类似Cursor的自动补全功能。

## 功能特性

1. **智能自动补全**：根据输入内容自动生成补全建议，类似Cursor的代码补全体验
2. **上下文管理**：智能读取历史聊天记录，构建对话上下文
3. **对话摘要**：长期对话中自动生成并维护关键信息摘要，解决上下文过长问题
   - 自动提取关键信息并保存到提示词中
   - 根据消息数量和时间阈值自动更新摘要
   - 保证长期聊天过程中的关键信息不丢失
4. **语言风格学习**：根据近期聊天记录学习发送方的语言风格
   - 分析词汇、句式、语气等特征
   - 在补全时应用学习到的风格
   - 支持多用户风格区分
5. **大模型集成**：通过Python脚本调用各种大模型API（OpenAI、Anthropic等）

## 项目结构

```
ChatRecommend/
├── cmd/
│   └── server/          # 主程序入口
├── internal/
│   ├── api/             # API接口层
│   ├── autocomplete/    # 自动补全引擎
│   ├── context/         # 上下文管理器
│   ├── style/           # 语言风格学习
│   ├── summary/         # 对话摘要生成
│   ├── llm/             # 大模型调用接口
│   ├── config/          # 配置管理
│   └── models/          # 数据模型
├── python/
│   └── llm_client.py    # Python大模型客户端
├── data/                # 数据目录
├── logs/                # 日志目录
├── config.yaml          # 配置文件
└── go.mod
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置

编辑 `config.yaml`，设置大模型API密钥和参数。

### 3. 安装Python依赖

```bash
pip install -r python/requirements.txt
```

### 4. 运行

```bash
go run cmd/server/main.go
```

## API接口

### HTTP接口

#### 获取补全建议
```bash
POST /api/chat/complete
Content-Type: application/json

{
  "conversation_id": "conv_123",
  "sender_id": "user_456",
  "input": "今天天气",
  "max_suggestions": 3
}
```

响应：
```json
{
  "suggestions": ["今天天气不错", "今天天气很好", "今天天气晴朗"],
  "context_used": "..."
}
```

#### 保存消息
```bash
POST /api/chat/message
Content-Type: application/json

{
  "conversation_id": "conv_123",
  "sender_id": "user_456",
  "content": "今天天气不错",
  "message_type": "text",
  "sequence": 1234567890
}
```

#### 获取聊天历史
```bash
GET /api/chat/history/:conversation_id?limit=50
```

### WebSocket接口

连接地址：`ws://localhost:8080/ws`

发送消息格式：
```json
{
  "type": "autocomplete",
  "autocomplete_request": {
    "conversation_id": "conv_123",
    "sender_id": "user_456",
    "input": "今天天气"
  }
}
```

接收消息格式：
```json
{
  "type": "autocomplete_response",
  "data": {
    "suggestions": ["今天天气不错", "今天天气很好"],
    "context_used": "..."
  }
}
```

## 配置说明

### 核心配置项

#### 对话摘要配置（summary）
- `update_threshold_messages`: 达到此消息数量后触发摘要更新（默认100）
- `update_threshold_hours`: 达到此时间后触发摘要更新（默认24小时）
- `max_summary_tokens`: 摘要最大长度（默认500 tokens）
- `key_info_count`: 关键信息提取数量（默认10）
- `auto_update`: 是否启用自动摘要（默认true）

#### 语言风格学习配置（style）
- `learning_messages_count`: 用于风格学习的近期消息数量（默认50）
- `update_threshold_messages`: 风格更新阈值（默认20条消息）
- `enabled`: 是否启用风格学习（默认true）

#### 上下文配置（context）
- `max_context_tokens`: 最大上下文长度（默认4000 tokens）
- `recent_messages_count`: 近期消息数量（默认50）
- `history_retention_count`: 保留的历史消息数量（默认1000）

### 工作原理

1. **对话摘要机制**：
   - 系统会定期分析对话内容，提取关键信息
   - 生成一个包含关键信息的提示词，用于后续对话上下文
   - 当消息数量或时间达到阈值时，自动更新摘要
   - 摘要会包含对话主题、关键决策、重要信息等

2. **语言风格学习**：
   - 系统分析用户近期消息，提取语言特征
   - 包括常用词汇、句子长度、语气、emoji使用等
   - 在生成补全建议时，会参考学习到的风格特征
   - 支持多用户，每个用户的风格独立学习

3. **上下文构建**：
   - 结合对话摘要（长期关键信息）
   - 结合用户语言风格（个性化特征）
   - 结合近期消息（最新对话内容）
   - 智能截断，确保不超过token限制

详见 `config.yaml` 文件中的注释。

## 潜在问题和解决方案

### 1. 对话摘要相关问题
- **问题**：如何准确提取关键信息？
  - **方案**：使用大模型分析对话，提取主题、决策、重要事实等
- **问题**：摘要更新频率如何平衡？
  - **方案**：基于消息数量和时间双重阈值，可配置
- **问题**：如何避免信息丢失？
  - **方案**：摘要包含关键信息，历史消息保留在数据库中

### 2. 语言风格学习问题
- **问题**：如何量化语言风格？
  - **方案**：分析词汇频率、句子长度、语气、标点使用等特征
- **问题**：如何区分不同用户？
  - **方案**：每个用户独立存储风格特征
- **问题**：风格变化如何处理？
  - **方案**：定期更新风格特征，基于近期消息

### 3. 性能优化
- 使用请求去抖（debounce）减少大模型调用
- 异步更新摘要和风格，不阻塞主流程
- 上下文智能截断，优先保留重要信息

### 4. 大模型调用
- 支持超时控制
- 错误处理和重试机制
- 支持多种大模型（OpenAI、Anthropic等）

## 开发计划

- [ ] 支持更多大模型（Claude、本地模型等）
- [ ] 改进摘要生成算法
- [ ] 优化风格学习算法
- [ ] 添加缓存机制
- [ ] 支持多语言
- [ ] 添加监控和指标

## 许可证

MIT

