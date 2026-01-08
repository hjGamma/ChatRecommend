package main

import (
	"fmt"
	"log"

	"ChatRecommend/internal/api"
	"ChatRecommend/internal/autocomplete"
	"ChatRecommend/internal/config"
	"ChatRecommend/internal/context"
	"ChatRecommend/internal/llm"
	"ChatRecommend/internal/models"
	"ChatRecommend/internal/style"
	"ChatRecommend/internal/summary"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	if err := config.InitLogger(&cfg.Log); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	logrus.Info("正在启动ChatRecommend服务...")

	// 初始化数据库
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 初始化大模型客户端
	llmClient := llm.NewClient(&cfg.LLM)

	// 初始化摘要管理器
	summaryLLMAdapter := summary.NewLLMAdapter(llmClient)
	summaryMgr := summary.NewManager(db, &cfg.Summary, summaryLLMAdapter)

	// 初始化风格管理器
	styleMgr := style.NewManager(db, &cfg.Style)

	// 初始化上下文管理器
	contextMgr := context.NewManager(db, &cfg.Context, summaryMgr, styleMgr)

	// 初始化自动补全引擎
	autocompleteEngine := autocomplete.NewEngine(db, &cfg.Autocomplete, contextMgr, llmClient)

	// 初始化API处理器
	handler := api.NewHandler(db, autocompleteEngine, summaryMgr, styleMgr)

	// 设置Gin模式
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建HTTP路由
	router := gin.Default()

	// CORS中间件
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API路由
	apiGroup := router.Group("/api")
	{
		chatGroup := apiGroup.Group("/chat")
		{
			chatGroup.POST("/complete", handler.Complete)
			chatGroup.POST("/message", handler.SaveMessage)
			chatGroup.GET("/history/:conversation_id", handler.GetHistory)
		}
	}

	// WebSocket路由
	router.GET("/ws", handler.HandleWebSocket)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 启动HTTP服务器
	httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	logrus.Infof("HTTP服务器启动在端口 %d", cfg.Server.HTTPPort)
	if err := router.Run(httpAddr); err != nil {
		log.Fatalf("启动HTTP服务器失败: %v", err)
	}
}

// initDatabase 初始化数据库
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Database.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&models.Conversation{},
		&models.Message{},
		&models.Summary{},
		&models.Style{},
	); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	logrus.Info("数据库初始化成功")
	return db, nil
}

