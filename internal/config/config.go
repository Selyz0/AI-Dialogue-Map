package config

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/spf13/viper"
)

//go:embed secret.toml
var secretTOMLContent []byte

// Config はアプリケーションの設定を保持します。
type Config struct {
	GeminiAPIKey string `mapstructure:"gemini_api_key"`
}

var Cfg Config

// loadConfig は設定ファイルから設定を読み込みます。
func LoadConfig() error {
	v := viper.New()
	v.SetConfigType("toml") // 設定ファイルの形式
	if err := v.ReadConfig(bytes.NewReader(secretTOMLContent)); err != nil {
		return fmt.Errorf("failed to read embedded config: %w", err)
	}

	if err := v.Unmarshal(&Cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}
