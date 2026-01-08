package api

import (
	"net/http"
	"strconv"
	"time"

	"ChatRecommend/internal/autocomplete"
	"ChatRecommend/internal/models"
	"ChatRecommend/internal/style"
	"ChatRecommend/internal/summary"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Handler API处理器
type Handler struct {
	db          *gorm.DB
	autocomplete *autocomplete.Engine
	summary     *summary.Manager
	style       *style.Manager
}

// NewHandler 创建API处理器
func NewHandler(db *gorm.DB, autocompleteEngine *autocomplete.Engine, summaryMgr *summary.Manager, styleMgr *style.Manager) *Handler {
	return &Handler{
		db:          db,
		autocomplete: autocompleteEngine,
		summary:     summaryMgr,
		style:       styleMgr,
	}
}

// Complete 获取补全建议
func (h *Handler) Complete(c *gin.Context) {
	var req models.AutocompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.autocomplete.GetSuggestions(&req)
	if err != nil {
		logrus.WithError(err).Error("获取补全建议失败")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SaveMessage 保存消息
func (h *Handler) SaveMessage(c *gin.Context) {
	var req models.SaveMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取或创建对话
	var conversation models.Conversation
	err := h.db.Where("conversation_id = ?", req.ConversationID).First(&conversation).Error
	if err == gorm.ErrRecordNotFound {
		conversation = models.Conversation{
			ConversationID: req.ConversationID,
			Participants:   "[]",
			LastMessageAt:  time.Now(),
		}
		if err := h.db.Create(&conversation).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建对话失败"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询对话失败"})
		return
	}

	// 创建消息
	message := models.Message{
		ConversationID: conversation.ID,
		SenderID:       req.SenderID,
		Content:        req.Content,
		MessageType:    req.MessageType,
		Sequence:       req.Sequence,
	}
	if message.MessageType == "" {
		message.MessageType = "text"
	}
	if message.Sequence == 0 {
		message.Sequence = time.Now().UnixNano()
	}

	if err := h.db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败"})
		return
	}

	// 更新对话最后消息时间
	conversation.LastMessageAt = time.Now()
	h.db.Save(&conversation)

	// 异步更新摘要和风格
	go h.updateSummaryAndStyle(conversation.ID, req.SenderID)

	c.JSON(http.StatusOK, gin.H{
		"message_id": message.ID,
		"status":     "success",
	})
}

// GetHistory 获取聊天历史
func (h *Handler) GetHistory(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id不能为空"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	var conversation models.Conversation
	if err := h.db.Where("conversation_id = ?", conversationID).First(&conversation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "对话不存在"})
		return
	}

	var messages []models.Message
	if err := h.db.Where("conversation_id = ?", conversation.ID).
		Order("sequence ASC, created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversation_id": conversationID,
		"messages":       messages,
	})
}

// updateSummaryAndStyle 异步更新摘要和风格
func (h *Handler) updateSummaryAndStyle(conversationID uint, senderID string) {
	// 获取所有消息
	var messages []models.Message
	if err := h.db.Where("conversation_id = ?", conversationID).
		Order("sequence ASC, created_at ASC").
		Find(&messages).Error; err != nil {
		logrus.WithError(err).Error("查询消息失败")
		return
	}

	// 更新摘要
	summary, err := h.summary.GetOrCreateSummary(conversationID)
	if err == nil && h.summary.ShouldUpdateSummary(summary, int64(len(messages))) {
		if err := h.summary.UpdateSummary(conversationID, messages); err != nil {
			logrus.WithError(err).Error("更新摘要失败")
		}
	}

	// 更新风格
	style, err := h.style.GetOrCreateStyle(conversationID, senderID)
	if err == nil && h.style.ShouldUpdateStyle(style, int64(len(messages))) {
		if err := h.style.UpdateStyle(conversationID, senderID, messages); err != nil {
			logrus.WithError(err).Error("更新风格失败")
		}
	}
}

