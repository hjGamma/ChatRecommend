package models

import (
	"time"

	"gorm.io/gorm"
)

// Conversation 对话模型
type Conversation struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 对话标识（如微信的会话ID、抖音的会话ID等）
	ConversationID string `gorm:"uniqueIndex;not null" json:"conversation_id"`
	// 参与者列表（JSON格式存储）
	Participants   string `gorm:"type:text" json:"participants"`
	// 最后一条消息时间
	LastMessageAt  time.Time `json:"last_message_at"`

	// 关联关系
	Messages []Message `gorm:"foreignKey:ConversationID;references:ID" json:"messages,omitempty"`
	Summary  *Summary  `gorm:"foreignKey:ConversationID;references:ID" json:"summary,omitempty"`
	Styles   []Style   `gorm:"foreignKey:ConversationID;references:ID" json:"styles,omitempty"`
}

// Message 消息模型
type Message struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 所属对话ID
	ConversationID uint   `gorm:"index;not null" json:"conversation_id"`
	// 消息发送者ID
	SenderID       string `gorm:"index;not null" json:"sender_id"`
	// 消息内容
	Content        string `gorm:"type:text;not null" json:"content"`
	// 消息类型（text, image, file等）
	MessageType    string `gorm:"default:text" json:"message_type"`
	// 消息序号（用于排序）
	Sequence       int64  `gorm:"index" json:"sequence"`
}

// Summary 对话摘要模型
type Summary struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 所属对话ID
	ConversationID uint   `gorm:"uniqueIndex;not null" json:"conversation_id"`
	// 摘要提示词（包含关键信息）
	Prompt         string `gorm:"type:text;not null" json:"prompt"`
	// 关键信息（JSON格式存储）
	KeyInfo        string `gorm:"type:text" json:"key_info"`
	// 最后更新时的消息数量
	LastMessageCount int64 `json:"last_message_count"`
	// 最后更新时间
	LastUpdatedAt    time.Time `json:"last_updated_at"`
	// 版本号（用于追踪更新）
	Version          int       `gorm:"default:1" json:"version"`
}

// Style 语言风格模型
type Style struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 所属对话ID
	ConversationID uint   `gorm:"index;not null" json:"conversation_id"`
	// 用户ID（消息发送者）
	UserID          string `gorm:"index;not null" json:"user_id"`
	// 风格特征（JSON格式存储）
	Features        string `gorm:"type:text;not null" json:"features"`
	// 风格描述（文本描述）
	Description     string `gorm:"type:text" json:"description"`
	// 最后更新时的消息数量
	LastMessageCount int64 `json:"last_message_count"`
	// 最后更新时间
	LastUpdatedAt    time.Time `json:"last_updated_at"`
}

// AutocompleteRequest 自动补全请求
type AutocompleteRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	SenderID       string `json:"sender_id" binding:"required"`
	Input          string `json:"input" binding:"required"`
	MaxSuggestions int    `json:"max_suggestions,omitempty"`
}

// AutocompleteResponse 自动补全响应
type AutocompleteResponse struct {
	Suggestions []string `json:"suggestions"`
	ContextUsed string   `json:"context_used,omitempty"`
}

// SaveMessageRequest 保存消息请求
type SaveMessageRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
	SenderID       string `json:"sender_id" binding:"required"`
	Content        string `json:"content" binding:"required"`
	MessageType    string `json:"message_type,omitempty"`
	Sequence       int64  `json:"sequence,omitempty"`
}

