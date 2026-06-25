package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Wechat   WechatConfig   `mapstructure:"wechat"`
	Spider   SpiderConfig   `mapstructure:"spider"`
	Bet      BetConfig      `mapstructure:"bet"`
	AI       AIConfig       `mapstructure:"ai"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

func (d DatabaseConfig) DSN() string {
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.DBName, d.SSLMode)
	if d.Password != "" {
		dsn += fmt.Sprintf(" password=%s", d.Password)
	}
	return dsn
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type WechatConfig struct {
	AppID  string `mapstructure:"appid"`
	Secret string `mapstructure:"secret"`
}

type SpiderConfig struct {
	IntervalOdds   time.Duration `mapstructure:"interval_odds"`
	IntervalResult time.Duration `mapstructure:"interval_result"`
	RetryCount     int           `mapstructure:"retry_count"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
	UserAgent      string        `mapstructure:"user_agent"`
	Referer        string        `mapstructure:"referer"`
	HealthFile     string        `mapstructure:"health_file"`
	HealthMaxAge   time.Duration `mapstructure:"health_max_age"`
}

type BetConfig struct {
	LockMinutesBefore int `mapstructure:"lock_minutes_before"`
	InitialCredits    int `mapstructure:"initial_credits"`
}

type AIConfig struct {
	Provider  string `mapstructure:"provider"`
	LLMAPIKey string `mapstructure:"llm_api_key"`
	LLMBaseURL string `mapstructure:"llm_base_url"`
	LLMModel   string `mapstructure:"llm_model"`
}

var cfg *Config

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	bindEnvVars(v)

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyEnvOverrides(&c)

	if c.Server.Mode == "release" && c.JWT.Secret == "sport_game2_jwt_secret_change_me" {
		return nil, fmt.Errorf("JWT secret must be changed from default in release mode")
	}

	cfg = &c
	return &c, nil
}

func bindEnvVars(v *viper.Viper) {
	envMap := map[string]string{
		"server.port":         "SERVER_PORT",
		"server.mode":         "SERVER_MODE",
		"database.host":       "DB_HOST",
		"database.port":       "DB_PORT",
		"database.user":       "DB_USER",
		"database.password":   "DB_PASSWORD",
		"database.dbname":     "DB_NAME",
		"database.sslmode":    "DB_SSLMODE",
		"jwt.secret":          "JWT_SECRET",
		"wechat.appid":        "WECHAT_APPID",
		"wechat.secret":       "WECHAT_SECRET",
		"ai.provider":         "AI_PROVIDER",
		"ai.llm_api_key":      "LLM_API_KEY",
		"ai.llm_base_url":     "LLM_BASE_URL",
		"ai.llm_model":        "LLM_MODEL",
	}
	for k, env := range envMap {
		_ = v.BindEnv(k, env)
	}
}

func applyEnvOverrides(c *Config) {
	if val := os.Getenv("DB_HOST"); val != "" {
		c.Database.Host = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			c.Database.Port = port
		}
	}
	if val := os.Getenv("DB_USER"); val != "" {
		c.Database.User = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		c.Database.Password = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		c.Database.DBName = val
	}
	if val := os.Getenv("DB_SSLMODE"); val != "" {
		c.Database.SSLMode = val
	}
	if val := os.Getenv("SERVER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			c.Server.Port = port
		}
	}
	if val := os.Getenv("SERVER_MODE"); val != "" {
		c.Server.Mode = val
	}
	if val := os.Getenv("JWT_SECRET"); val != "" {
		c.JWT.Secret = val
	}
	if val := os.Getenv("WECHAT_APPID"); val != "" {
		c.Wechat.AppID = val
	}
	if val := os.Getenv("WECHAT_SECRET"); val != "" {
		c.Wechat.Secret = val
	}
}

func Get() *Config {
	if cfg == nil {
		c, _ := Load()
		return c
	}
	return cfg
}
