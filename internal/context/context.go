package context

import (
	"fmt"
	"strings"

	"ChatRecommend/internal/config"
	"ChatRecommend/internal/models"
	"ChatRecommend/internal/style"
	"ChatRecommend/internal/summary"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Manager 上下文管理器
type Manager struct {
	db       *gorm.DB
	config   *config.ContextConfig
	summary  *summary.Manager
	style    *style.Manager
}

// NewManager 创建上下文管理器
func NewManager(db *gorm.DB, cfg *config.ContextConfig, summaryMgr *summary.Manager, styleMgr *style.Manager) *Manager {
	return &Manager{
		db:      db,
		config:  cfg,
		summary: summaryMgr,
		style:   styleMgr,
	}
}

// BuildContext 构建对话上下文
func (m *Manager) BuildContext(conversationID uint, senderID string, currentInput string) (string, error) {
	var conversation models.Conversation
	if err := m.db.First(&conversation, conversationID).Error; err != nil {
		return "", fmt.Errorf("查询对话失败: %w", err)
	}

	// 1. 获取对话摘要提示词
	summaryPrompt, err := m.summary.GetSummaryPrompt(conversationID)
	if err != nil {
		logrus.WithError(err).Warn("获取摘要失败")
	}

	// 2. 获取用户语言风格提示词
	stylePrompt, err := m.style.GetStylePrompt(conversationID, senderID)
	if err != nil {
		logrus.WithError(err).Warn("获取风格失败")
	}

	// 3. 获取近期消息
	recentMessages, err := m.getRecentMessages(conversationID, m.config.RecentMessagesCount)
	if err != nil {
		return "", fmt.Errorf("获取近期消息失败: %w", err)
	}

	// 4. 构建完整上下文
	var contextBuilder strings.Builder

	// 添加摘要提示词
	if summaryPrompt != "" {
		contextBuilder.WriteString("=== 对话背景信息 ===\n")
		contextBuilder.WriteString(summaryPrompt)
		contextBuilder.WriteString("\n\n")
	}

	// 添加风格提示词
	if stylePrompt != "" {
		contextBuilder.WriteString("=== 用户语言风格 ===\n")
		contextBuilder.WriteString(stylePrompt)
		contextBuilder.WriteString("\n\n")
	}

	// 添加近期对话历史
	if len(recentMessages) > 0 {
		contextBuilder.WriteString("=== 近期对话历史 ===\n")
		for _, msg := range recentMessages {
			contextBuilder.WriteString(fmt.Sprintf("[%s]: %s\n", msg.SenderID, msg.Content))
		}
		contextBuilder.WriteString("\n")
	}

	// 添加当前输入
	contextBuilder.WriteString("=== 当前输入 ===\n")
	contextBuilder.WriteString(fmt.Sprintf("[%s]: %s", senderID, currentInput))

	context := contextBuilder.String()

	// 5. 检查并截断上下文（简单实现，实际应该按token计算）
	if len([]rune(context)) > m.config.MaxContextTokens*3 { // 粗略估算：1 token ≈ 3 字符
		context = truncateContext(context, m.config.MaxContextTokens*3)
		logrus.Warn("上下文已截断")
	}

	return context, nil
}

// getRecentMessages 获取近期消息
func (m *Manager) getRecentMessages(conversationID uint, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := m.db.Where("conversation_id = ?", conversationID).
		Order("sequence DESC, created_at DESC").
		Limit(limit).
		Find(&messages).Error
	
	if err != nil {
		return nil, err
	}

	// 反转顺序，使消息按时间正序排列
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// truncateContext 截断上下文（保留摘要和风格，截断历史消息）
func truncateContext(context string, maxLength int) string {
	if len([]rune(context)) <= maxLength {
		return context
	}

	// 找到"近期对话历史"部分
	historyStart := strings.Index(context, "=== 近期对话历史 ===")
	if historyStart == -1 {
		// 如果没有历史部分，直接截断
		runes := []rune(context)
		if len(runes) > maxLength {
			return string(runes[:maxLength]) + "..."
		}
		return context
	}

	// 保留摘要和风格部分
	prefix := context[:historyStart]
	history := context[historyStart:]

	// 计算可用长度
	prefixRunes := []rune(prefix)
	availableLength := maxLength - len(prefixRunes) - 100 // 预留一些空间

	if availableLength <= 0 {
		return prefix + "\n[上下文已截断]"
	}

	// 截断历史部分
	historyRunes := []rune(history)
	if len(historyRunes) > availableLength {
		historyRunes = historyRunes[:availableLength]
		history = string(historyRunes) + "\n[上下文已截断]"
	}

	return prefix + history
}

