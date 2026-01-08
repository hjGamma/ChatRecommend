# 使用示例

## 1. 基本使用流程

### 步骤1：启动服务

```bash
# 安装Go依赖
go mod download

# 安装Python依赖
pip install -r python/requirements.txt

# 配置API密钥（编辑config.yaml）
# 设置 llm.api.api_key 为你的OpenAI API密钥

# 启动服务
go run cmd/server/main.go
```

### 步骤2：创建对话并发送消息

```bash
# 保存第一条消息（会自动创建对话）
curl -X POST http://localhost:8080/api/chat/message \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "wechat_123456",
    "sender_id": "user_alice",
    "content": "你好，我是Alice，我想了解一下这个项目",
    "message_type": "text"
  }'
```

### 步骤3：获取补全建议

```bash
# 获取补全建议
curl -X POST http://localhost:8080/api/chat/complete \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "wechat_123456",
    "sender_id": "user_alice",
    "input": "这个项目的主要功能是",
    "max_suggestions": 3
  }'
```

## 2. WebSocket实时补全

### JavaScript示例

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('WebSocket连接已建立');
  
  // 发送补全请求
  ws.send(JSON.stringify({
    type: 'autocomplete',
    autocomplete_request: {
      conversation_id: 'wechat_123456',
      sender_id: 'user_alice',
      input: '这个项目的主要功能是'
    }
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  if (response.type === 'autocomplete_response') {
    console.log('补全建议:', response.data.suggestions);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket错误:', error);
};
```

## 3. 集成到微信/抖音等应用

### 微信集成示例（伪代码）

```go
// 微信消息处理
func handleWeChatMessage(msg WeChatMessage) {
    // 1. 保存消息到系统
    saveMessageRequest := models.SaveMessageRequest{
        ConversationID: msg.ChatID,
        SenderID:       msg.FromUser,
        Content:        msg.Text,
        MessageType:    "text",
    }
    // 调用API保存消息
    
    // 2. 如果用户正在输入，获取补全建议
    if msg.IsTyping {
        autocompleteRequest := models.AutocompleteRequest{
            ConversationID: msg.ChatID,
            SenderID:       msg.FromUser,
            Input:          msg.PartialText,
        }
        // 调用API获取补全建议
        // 将建议发送回微信
    }
}
```

## 4. 摘要和风格学习示例

### 自动摘要更新

系统会在以下情况自动更新摘要：
- 消息数量达到 `summary.update_threshold_messages`（默认100条）
- 距离上次更新超过 `summary.update_threshold_hours`（默认24小时）

### 风格学习

系统会分析用户的语言特征：
- 常用词汇和短语
- 平均句子长度
- 语气（正式/随意/友好）
- Emoji使用频率

这些特征会在生成补全建议时应用。

## 5. 配置示例

### 开发环境配置

```yaml
llm:
  model_type: "openai"
  api:
    model: "gpt-3.5-turbo"  # 使用更便宜的模型
    temperature: 0.7
    max_tokens: 1000

summary:
  update_threshold_messages: 50  # 更频繁更新
  auto_update: true

style:
  learning_messages_count: 30
  enabled: true
```

### 生产环境配置

```yaml
llm:
  model_type: "openai"
  api:
    model: "gpt-4"  # 使用更强大的模型
    temperature: 0.7
    max_tokens: 2000

summary:
  update_threshold_messages: 100
  update_threshold_hours: 24
  max_summary_tokens: 500

style:
  learning_messages_count: 50
  update_threshold_messages: 20
```

