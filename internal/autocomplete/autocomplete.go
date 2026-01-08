package autocomplete

import (
	"fmt"
	"sync"
	"time"

	"ChatRecommend/internal/config"
	"ChatRecommend/internal/context"
	"ChatRecommend/internal/llm"
	"ChatRecommend/internal/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Engine 自动补全引擎
type Engine struct {
	db          *gorm.DB
	config      *config.AutocompleteConfig
	contextMgr  *context.Manager
	llmClient   *llm.Client
	debounceMap sync.Map // 用于请求去抖
}

// NewEngine 创建自动补全引擎
func NewEngine(db *gorm.DB, cfg *config.AutocompleteConfig, contextMgr *context.Manager, llmClient *llm.Client) *Engine {
	return &Engine{
		db:         db,
		config:     cfg,
		contextMgr: contextMgr,
		llmClient:  llmClient,
	}
}

// GetSuggestions 获取补全建议
func (e *Engine) GetSuggestions(req *models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	// 检查输入长度
	if len([]rune(req.Input)) < e.config.MinTriggerLength {
		return &models.AutocompleteResponse{
			Suggestions: []string{},
		}, nil
	}

	// 获取对话ID（通过conversation_id字符串查找）
	var conversation models.Conversation
	if err := e.db.Where("conversation_id = ?", req.ConversationID).First(&conversation).Error; err != nil {
		return nil, fmt.Errorf("查询对话失败: %w", err)
	}

	// 构建上下文
	ctx, err := e.contextMgr.BuildContext(conversation.ID, req.SenderID, req.Input)
	if err != nil {
		return nil, fmt.Errorf("构建上下文失败: %w", err)
	}

	// 调用大模型生成补全建议
	maxSuggestions := e.config.SuggestionCount
	if req.MaxSuggestions > 0 {
		maxSuggestions = req.MaxSuggestions
	}

	suggestions, err := e.llmClient.Complete(ctx, req.Input)
	if err != nil {
		return nil, fmt.Errorf("生成补全建议失败: %w", err)
	}

	// 限制建议数量
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	logrus.WithFields(logrus.Fields{
		"conversation_id": req.ConversationID,
		"input_length":    len(req.Input),
		"suggestions":     len(suggestions),
	}).Debug("生成补全建议")

	return &models.AutocompleteResponse{
		Suggestions: suggestions,
		ContextUsed: ctx,
	}, nil
}

// GetSuggestionsWithDebounce 带去抖的获取补全建议
func (e *Engine) GetSuggestionsWithDebounce(req *models.AutocompleteRequest) (*models.AutocompleteResponse, error) {
	// 生成去抖键
	debounceKey := fmt.Sprintf("%s:%s", req.ConversationID, req.SenderID)

	// 检查是否有正在进行的请求
	if existing, ok := e.debounceMap.Load(debounceKey); ok {
		if timer, ok := existing.(*time.Timer); ok {
			timer.Stop()
		}
	}

	// 创建结果通道
	resultChan := make(chan *models.AutocompleteResponse, 1)
	errorChan := make(chan error, 1)

	// 设置去抖定时器
	timer := time.AfterFunc(time.Duration(e.config.DebounceMs)*time.Millisecond, func() {
		resp, err := e.GetSuggestions(req)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- resp
		}
		e.debounceMap.Delete(debounceKey)
	})

	e.debounceMap.Store(debounceKey, timer)

	// 等待结果
	select {
	case resp := <-resultChan:
		return resp, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(time.Duration(e.config.DebounceMs)*2*time.Millisecond + 5*time.Second):
		return nil, fmt.Errorf("获取补全建议超时")
	}
}

