package style

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ChatRecommend/internal/config"
	"ChatRecommend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Manager 风格管理器
type Manager struct {
	db     *gorm.DB
	config *config.StyleConfig
}

// StyleFeatures 风格特征
type StyleFeatures struct {
	Vocabulary      map[string]int `json:"vocabulary"`       // 常用词汇及频率
	SentenceLength  float64        `json:"sentence_length"`  // 平均句子长度
	EmojiUsage      float64        `json:"emoji_usage"`      // emoji使用频率
	Tone            string         `json:"tone"`             // 语气（formal, casual, friendly等）
	Punctuation     map[string]int `json:"punctuation"`      // 标点符号使用
	CommonPhrases   []string       `json:"common_phrases"`   // 常用短语
}

// NewManager 创建风格管理器
func NewManager(db *gorm.DB, cfg *config.StyleConfig) *Manager {
	return &Manager{
		db:     db,
		config: cfg,
	}
}

// GetOrCreateStyle 获取或创建用户风格
func (m *Manager) GetOrCreateStyle(conversationID uint, userID string) (*models.Style, error) {
	var style models.Style
	err := m.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&style).Error
	if err == nil {
		return &style, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询风格失败: %w", err)
	}

	// 创建新风格
	style = models.Style{
		ConversationID:  conversationID,
		UserID:          userID,
		Features:        "{}",
		Description:     "",
		LastMessageCount: 0,
		LastUpdatedAt:   time.Now(),
	}
	if err := m.db.Create(&style).Error; err != nil {
		return nil, fmt.Errorf("创建风格失败: %w", err)
	}

	return &style, nil
}

// ShouldUpdateStyle 判断是否需要更新风格
func (m *Manager) ShouldUpdateStyle(style *models.Style, currentMessageCount int64) bool {
	if !m.config.Enabled {
		return false
	}

	// 检查消息数量阈值
	if currentMessageCount-style.LastMessageCount >= int64(m.config.UpdateThresholdMessages) {
		return true
	}

	return false
}

// UpdateStyle 更新用户语言风格
func (m *Manager) UpdateStyle(conversationID uint, userID string, messages []models.Message) error {
	if !m.config.Enabled {
		return nil
	}

	// 过滤出该用户的消息
	userMessages := make([]models.Message, 0)
	for _, msg := range messages {
		if msg.SenderID == userID {
			userMessages = append(userMessages, msg)
		}
	}

	if len(userMessages) == 0 {
		return nil
	}

	// 分析风格特征
	features := m.analyzeStyle(userMessages)
	description := m.generateDescription(features)

	// 序列化特征
	featuresJSON, err := json.Marshal(features)
	if err != nil {
		return fmt.Errorf("序列化风格特征失败: %w", err)
	}

	// 更新或创建风格记录
	style, err := m.GetOrCreateStyle(conversationID, userID)
	if err != nil {
		return err
	}

	style.Features = string(featuresJSON)
	style.Description = description
	style.LastMessageCount = int64(len(userMessages))
	style.LastUpdatedAt = time.Now()

	if err := m.db.Save(style).Error; err != nil {
		return fmt.Errorf("保存风格失败: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"user_id":         userID,
	}).Info("用户语言风格已更新")

	return nil
}

// GetStyleFeatures 获取用户风格特征
func (m *Manager) GetStyleFeatures(conversationID uint, userID string) (*StyleFeatures, error) {
	style, err := m.GetOrCreateStyle(conversationID, userID)
	if err != nil {
		return nil, err
	}

	var features StyleFeatures
	if style.Features != "" && style.Features != "{}" {
		if err := json.Unmarshal([]byte(style.Features), &features); err != nil {
			logrus.WithError(err).Warn("解析风格特征失败")
			return &StyleFeatures{}, nil
		}
	}

	return &features, nil
}

// GetStylePrompt 获取风格提示词（用于大模型）
func (m *Manager) GetStylePrompt(conversationID uint, userID string) (string, error) {
	features, err := m.GetStyleFeatures(conversationID, userID)
	if err != nil {
		return "", err
	}

	if features == nil || len(features.Vocabulary) == 0 {
		return "", nil
	}

	// 构建风格提示词
	var prompt strings.Builder
	prompt.WriteString("用户的语言风格特征：\n")
	
	if features.Tone != "" {
		prompt.WriteString(fmt.Sprintf("- 语气：%s\n", features.Tone))
	}
	
	if features.SentenceLength > 0 {
		prompt.WriteString(fmt.Sprintf("- 平均句子长度：%.1f字\n", features.SentenceLength))
	}
	
	if len(features.CommonPhrases) > 0 {
		prompt.WriteString(fmt.Sprintf("- 常用短语：%s\n", strings.Join(features.CommonPhrases[:min(5, len(features.CommonPhrases))], "、")))
	}

	return prompt.String(), nil
}

// analyzeStyle 分析消息风格特征
func (m *Manager) analyzeStyle(messages []models.Message) *StyleFeatures {
	features := &StyleFeatures{
		Vocabulary:    make(map[string]int),
		Punctuation:   make(map[string]int),
		CommonPhrases: make([]string, 0),
	}

	totalLength := 0
	emojiCount := 0
	totalChars := 0

	// 常用词汇（简单实现，可以改进）
	wordFreq := make(map[string]int)

	for _, msg := range messages {
		content := msg.Content
		totalChars += len([]rune(content))
		
		// 统计句子长度
		sentences := strings.Split(content, "。")
		for _, s := range sentences {
			if len(s) > 0 {
				totalLength += len([]rune(s))
			}
		}

		// 统计emoji（简单判断，可以改进）
		for _, r := range content {
			if r >= 0x1F300 && r <= 0x1F9FF {
				emojiCount++
			}
		}

		// 统计标点符号
		for _, r := range content {
			if strings.ContainsRune("，。！？、；：", r) {
				features.Punctuation[string(r)]++
			}
		}

		// 简单分词（可以改进为更专业的分词）
		words := strings.Fields(content)
		for _, word := range words {
			if len([]rune(word)) >= 2 {
				wordFreq[word]++
			}
		}
	}

	// 计算平均句子长度
	sentenceCount := 0
	for _, msg := range messages {
		sentenceCount += len(strings.Split(msg.Content, "。"))
	}
	if sentenceCount > 0 {
		features.SentenceLength = float64(totalLength) / float64(sentenceCount)
	}

	// 计算emoji使用频率
	if totalChars > 0 {
		features.EmojiUsage = float64(emojiCount) / float64(totalChars) * 100
	}

	// 获取最常用的词汇
	topWords := getTopN(wordFreq, 10)
	for word, count := range topWords {
		features.Vocabulary[word] = count
	}

	// 判断语气（简单实现）
	if features.SentenceLength < 10 && features.EmojiUsage > 2 {
		features.Tone = "casual"
	} else if features.SentenceLength > 30 {
		features.Tone = "formal"
	} else {
		features.Tone = "friendly"
	}

	return features
}

// generateDescription 生成风格描述
func (m *Manager) generateDescription(features *StyleFeatures) string {
	var desc strings.Builder
	
	desc.WriteString(fmt.Sprintf("语气：%s，", features.Tone))
	desc.WriteString(fmt.Sprintf("平均句子长度：%.1f字，", features.SentenceLength))
	
	if features.EmojiUsage > 2 {
		desc.WriteString("经常使用表情符号，")
	}
	
	if len(features.CommonPhrases) > 0 {
		desc.WriteString(fmt.Sprintf("常用短语：%s", strings.Join(features.CommonPhrases[:min(3, len(features.CommonPhrases))], "、")))
	}

	return desc.String()
}

// getTopN 获取频率最高的N个词
func getTopN(wordFreq map[string]int, n int) map[string]int {
	// 简单实现，可以改进为堆排序
	result := make(map[string]int)
	
	type wordCount struct {
		word  string
		count int
	}
	
	words := make([]wordCount, 0, len(wordFreq))
	for word, count := range wordFreq {
		words = append(words, wordCount{word, count})
	}
	
	// 简单排序
	for i := 0; i < len(words)-1 && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(words); j++ {
			if words[j].count > words[maxIdx].count {
				maxIdx = j
			}
		}
		words[i], words[maxIdx] = words[maxIdx], words[i]
		result[words[i].word] = words[i].count
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

