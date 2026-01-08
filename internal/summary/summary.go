package summary

import (
	"encoding/json"
	"fmt"
	"time"

	"ChatRecommend/internal/config"
	"ChatRecommend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Manager 摘要管理器
type Manager struct {
	db     *gorm.DB
	config *config.SummaryConfig
	llm    LLMInterface
}

// LLMInterface 大模型接口（用于生成摘要）
type LLMInterface interface {
	GenerateSummary(messages []models.Message, existingSummary *models.Summary) (string, string, error)
}

// NewManager 创建摘要管理器
func NewManager(db *gorm.DB, cfg *config.SummaryConfig, llm LLMInterface) *Manager {
	return &Manager{
		db:     db,
		config: cfg,
		llm:    llm,
	}
}

// GetOrCreateSummary 获取或创建对话摘要
func (m *Manager) GetOrCreateSummary(conversationID uint) (*models.Summary, error) {
	var summary models.Summary
	err := m.db.Where("conversation_id = ?", conversationID).First(&summary).Error
	if err == nil {
		return &summary, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询摘要失败: %w", err)
	}

	// 创建新摘要
	summary = models.Summary{
		ConversationID:  conversationID,
		Prompt:          "",
		KeyInfo:         "[]",
		LastMessageCount: 0,
		LastUpdatedAt:   time.Now(),
		Version:         1,
	}
	if err := m.db.Create(&summary).Error; err != nil {
		return nil, fmt.Errorf("创建摘要失败: %w", err)
	}

	return &summary, nil
}

// ShouldUpdateSummary 判断是否需要更新摘要
func (m *Manager) ShouldUpdateSummary(summary *models.Summary, currentMessageCount int64) bool {
	if !m.config.AutoUpdate {
		return false
	}

	// 检查消息数量阈值
	if currentMessageCount-summary.LastMessageCount >= int64(m.config.UpdateThresholdMessages) {
		return true
	}

	// 检查时间阈值
	if time.Since(summary.LastUpdatedAt) >= time.Duration(m.config.UpdateThresholdHours)*time.Hour {
		return true
	}

	return false
}

// UpdateSummary 更新对话摘要
func (m *Manager) UpdateSummary(conversationID uint, messages []models.Message) error {
	summary, err := m.GetOrCreateSummary(conversationID)
	if err != nil {
		return err
	}

	// 调用大模型生成摘要
	prompt, keyInfo, err := m.llm.GenerateSummary(messages, summary)
	if err != nil {
		return fmt.Errorf("生成摘要失败: %w", err)
	}

	// 更新摘要
	summary.Prompt = prompt
	summary.KeyInfo = keyInfo
	summary.LastMessageCount = int64(len(messages))
	summary.LastUpdatedAt = time.Now()
	summary.Version++

	if err := m.db.Save(summary).Error; err != nil {
		return fmt.Errorf("保存摘要失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"version":         summary.Version,
	}).Info("对话摘要已更新")

	return nil
}

// GetSummaryPrompt 获取摘要提示词
func (m *Manager) GetSummaryPrompt(conversationID uint) (string, error) {
	summary, err := m.GetOrCreateSummary(conversationID)
	if err != nil {
		return "", err
	}
	return summary.Prompt, nil
}

// GetKeyInfo 获取关键信息
func (m *Manager) GetKeyInfo(conversationID uint) ([]map[string]interface{}, error) {
	summary, err := m.GetOrCreateSummary(conversationID)
	if err != nil {
		return nil, err
	}

	var keyInfo []map[string]interface{}
	if summary.KeyInfo != "" && summary.KeyInfo != "[]" {
		if err := json.Unmarshal([]byte(summary.KeyInfo), &keyInfo); err != nil {
			logrus.WithError(err).Warn("解析关键信息失败")
			return []map[string]interface{}{}, nil
		}
	}

	return keyInfo, nil
}

