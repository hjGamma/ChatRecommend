package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	LLM          LLMConfig          `mapstructure:"llm"`
	Context      ContextConfig       `mapstructure:"context"`
	Summary      SummaryConfig       `mapstructure:"summary"`
	Style        StyleConfig         `mapstructure:"style"`
	Autocomplete AutocompleteConfig  `mapstructure:"autocomplete"`
	Server       ServerConfig        `mapstructure:"server"`
	Database     DatabaseConfig      `mapstructure:"database"`
	Log          LogConfig           `mapstructure:"log"`
}

// LLMConfig 大模型配置
type LLMConfig struct {
	PythonScript     string    `mapstructure:"python_script"`
	PythonInterpreter string   `mapstructure:"python_interpreter"`
	ModelType        string    `mapstructure:"model_type"`
	API              APIConfig `mapstructure:"api"`
	Timeout          int       `mapstructure:"timeout"`
}

// APIConfig API配置
type APIConfig struct {
	BaseURL          string  `mapstructure:"base_url"`
	APIKey           string  `mapstructure:"api_key"`
	Model            string  `mapstructure:"model"`
	Temperature      float64 `mapstructure:"temperature"`
	MaxTokens        int     `mapstructure:"max_tokens"`
	TopP             float64 `mapstructure:"top_p"`
	FrequencyPenalty float64 `mapstructure:"frequency_penalty"`
	PresencePenalty  float64 `mapstructure:"presence_penalty"`
}

// ContextConfig 上下文配置
type ContextConfig struct {
	MaxContextTokens    int `mapstructure:"max_context_tokens"`
	RecentMessagesCount int `mapstructure:"recent_messages_count"`
	HistoryRetentionCount int `mapstructure:"history_retention_count"`
}

// SummaryConfig 对话摘要配置
type SummaryConfig struct {
	UpdateThresholdMessages int  `mapstructure:"update_threshold_messages"`
	UpdateThresholdHours    int  `mapstructure:"update_threshold_hours"`
	MaxSummaryTokens        int  `mapstructure:"max_summary_tokens"`
	KeyInfoCount            int  `mapstructure:"key_info_count"`
	AutoUpdate              bool `mapstructure:"auto_update"`
}

// StyleConfig 语言风格学习配置
type StyleConfig struct {
	LearningMessagesCount int      `mapstructure:"learning_messages_count"`
	FeatureDimensions     []string `mapstructure:"feature_dimensions"`
	UpdateThresholdMessages int    `mapstructure:"update_threshold_messages"`
	Enabled               bool     `mapstructure:"enabled"`
}

// AutocompleteConfig 自动补全配置
type AutocompleteConfig struct {
	MinTriggerLength int `mapstructure:"min_trigger_length"`
	SuggestionCount  int `mapstructure:"suggestion_count"`
	DebounceMs       int `mapstructure:"debounce_ms"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort      int      `mapstructure:"http_port"`
	WSPort        int      `mapstructure:"ws_port"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DBPath  string `mapstructure:"db_path"`
	LogMode bool   `mapstructure:"log_mode"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}

var globalConfig *Config

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	viper.SetConfigType("yaml")
	
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// 默认查找当前目录和上级目录
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("..")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 确保数据目录存在
	if config.Database.DBPath != "" {
		dbDir := filepath.Dir(config.Database.DBPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
	}

	// 确保日志目录存在
	if config.Log.Output == "file" && config.Log.FilePath != "" {
		logDir := filepath.Dir(config.Log.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	globalConfig = config
	return config, nil
}

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if cfg.LLM.PythonScript == "" {
		return fmt.Errorf("python_script 不能为空")
	}
	if cfg.LLM.Timeout <= 0 {
		return fmt.Errorf("timeout 必须大于0")
	}
	if cfg.Context.MaxContextTokens <= 0 {
		return fmt.Errorf("max_context_tokens 必须大于0")
	}
	if cfg.Server.HTTPPort <= 0 {
		return fmt.Errorf("http_port 必须大于0")
	}
	if cfg.Server.WSPort <= 0 {
		return fmt.Errorf("ws_port 必须大于0")
	}
	return nil
}

// InitLogger 初始化日志
func InitLogger(cfg *LogConfig) error {
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}

	if cfg.Output == "file" && cfg.FilePath != "" {
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logrus.SetOutput(file)
	}

	return nil
}

