package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config はアプリケーションの設定を管理
type Config struct {
	ForceOverwrite      bool
	SpotifyClientID     string
	SpotifyClientSecret string
}

// NewConfig は新しい設定インスタンスを作成
func NewConfig(forceOverwrite bool) *Config {
	return &Config{
		ForceOverwrite: forceOverwrite,
	}
}

// LoadEnv は環境変数を読み込み
func (c *Config) LoadEnv() error {
	// .env.localファイルを読み込み
	err := godotenv.Load(".env.local")
	if err != nil {
		// .env.localが存在しない場合は.envを試行
		err = godotenv.Load()
		if err != nil {
			log.Printf("Warning: .env file not found: %v", err)
		}
	}

	// Spotify認証情報を取得
	c.SpotifyClientID = os.Getenv("SPOTIFY_CLIENT_ID")
	c.SpotifyClientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")

	return nil
}

// ValidateSpotifyCredentials はSpotify認証情報の有効性をチェック
func (c *Config) ValidateSpotifyCredentials() error {
	if c.SpotifyClientID == "" || c.SpotifyClientSecret == "" {
		return fmt.Errorf("Spotify認証情報が設定されていません。SPOTIFY_CLIENT_ID と SPOTIFY_CLIENT_SECRET 環境変数を設定してください")
	}
	return nil
}
