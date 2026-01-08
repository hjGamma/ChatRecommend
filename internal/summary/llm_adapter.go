package summary

import (
	"ChatRecommend/internal/models"
)

// LLMAdapter LLM接口适配器
type LLMAdapter struct {
	llmClient LLMInterface
}

// NewLLMAdapter 创建LLM适配器
func NewLLMAdapter(llmClient LLMInterface) *LLMAdapter {
	return &LLMAdapter{
		llmClient: llmClient,
	}
}

// GenerateSummary 实现LLMInterface接口
func (a *LLMAdapter) GenerateSummary(messages []models.Message, existingSummary *models.Summary) (string, string, error) {
	return a.llmClient.GenerateSummary(messages, existingSummary)
}

