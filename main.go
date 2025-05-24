package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// SpotifySearchResponse はSpotify検索APIのレスポンス構造体
type SpotifySearchResponse struct {
	Albums struct {
		Items []struct {
			Name   string `json:"name"`
			Images []struct {
				URL    string `json:"url"`
				Height int    `json:"height"`
				Width  int    `json:"width"`
			} `json:"images"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"albums"`
}

// MusicFileProcessor は音楽ファイルの処理を行う構造体
type MusicFileProcessor struct {
	spotifyToken string
	client       *http.Client
}

// NewMusicFileProcessor は新しいプロセッサーを作成
func NewMusicFileProcessor() *MusicFileProcessor {
	return &MusicFileProcessor{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// extractMetadata は音楽ファイルからメタデータを抽出
func (p *MusicFileProcessor) extractMetadata(filePath string) (artist, album, title string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", "", fmt.Errorf("ファイルを開けませんでした: %w", err)
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		return "", "", "", fmt.Errorf("メタデータを読み取れませんでした: %w", err)
	}

	return metadata.Artist(), metadata.Album(), metadata.Title(), nil
}

// getSpotifyToken はSpotify Web APIのアクセストークンを取得
// 注意: 実際の使用には client_id と client_secret が必要です
func (p *MusicFileProcessor) getSpotifyToken(clientID, clientSecret string) error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	p.spotifyToken = tokenResp.AccessToken
	return nil
}

// searchArtwork はSpotify APIを使用してアートワークを検索
func (p *MusicFileProcessor) searchArtwork(artist, album string) (string, error) {
	if p.spotifyToken == "" {
		return "", fmt.Errorf("Spotifyトークンが設定されていません")
	}

	query := fmt.Sprintf("artist:%s album:%s", artist, album)
	encodedQuery := url.QueryEscape(query)

	searchURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=album&limit=1", encodedQuery)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+p.spotifyToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var searchResp SpotifySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", err
	}

	if len(searchResp.Albums.Items) == 0 || len(searchResp.Albums.Items[0].Images) == 0 {
		return "", fmt.Errorf("アートワークが見つかりませんでした")
	}

	// 最高解像度の画像を選択
	bestImage := searchResp.Albums.Items[0].Images[0]
	for _, img := range searchResp.Albums.Items[0].Images {
		if img.Height > bestImage.Height {
			bestImage = img
		}
	}

	return bestImage.URL, nil
}

// downloadImage は指定されたURLから画像をダウンロード
func (p *MusicFileProcessor) downloadImage(imageURL, outputPath string) error {
	resp, err := p.client.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("画像のダウンロードに失敗: %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// embedArtwork はffmpegを使用してアートワークを埋め込み
func (p *MusicFileProcessor) embedArtwork(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:0",
		"-map", "1:0",
		"-c", "copy",
		"-id3v2_version", "3",
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", `comment=Cover (front)`,
		"-y", // 既存ファイルを上書き
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpegエラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// processFile は単一の音楽ファイルを処理
func (p *MusicFileProcessor) processFile(filePath string) error {
	fmt.Printf("処理中: %s\n", filePath)

	// メタデータを抽出
	artist, album, title, err := p.extractMetadata(filePath)
	if err != nil {
		return fmt.Errorf("メタデータ抽出エラー: %w", err)
	}

	fmt.Printf("  アーティスト: %s\n", artist)
	fmt.Printf("  アルバム: %s\n", album)
	fmt.Printf("  タイトル: %s\n", title)

	if artist == "" && album == "" {
		fmt.Printf("  警告: アーティストまたはアルバム情報が不足しています。スキップします。\n\n")
		return nil
	}

	// アートワークを検索
	fmt.Println("  アートワークを検索中...")
	artworkURL, err := p.searchArtwork(artist, album)
	if err != nil {
		fmt.Printf("  警告: アートワーク検索に失敗しました (%v)。スキップします。\n\n", err)
		return nil
	}

	// 一時ファイルパスを生成
	tempImagePath := filepath.Join(os.TempDir(), "temp_artwork.jpg")
	defer os.Remove(tempImagePath)

	// 画像をダウンロード
	fmt.Println("  アートワークをダウンロード中...")
	if err := p.downloadImage(artworkURL, tempImagePath); err != nil {
		return fmt.Errorf("画像ダウンロードエラー: %w", err)
	}

	// 一時出力ファイルパスを生成（元ファイルを上書きするため）
	tempOutputPath := filePath + ".tmp"

	// アートワークを埋め込み
	fmt.Println("  アートワークを埋め込み中...")
	if err := p.embedArtwork(filePath, tempImagePath, tempOutputPath); err != nil {
		return fmt.Errorf("アートワーク埋め込みエラー: %w", err)
	}

	// 元ファイルを一時ファイルで置き換え
	if err := os.Rename(tempOutputPath, filePath); err != nil {
		os.Remove(tempOutputPath) // クリーンアップ
		return fmt.Errorf("ファイル置き換えエラー: %w", err)
	}

	fmt.Printf("  完了: %s\n\n", filePath)
	return nil
}

// processDirectory はディレクトリ内の音楽ファイルを再帰的に処理
func (p *MusicFileProcessor) processDirectory(dirPath string) error {
	musicExtensions := map[string]bool{
		".mp3":  true,
		".m4a":  true,
		".flac": true,
		".wav":  true,
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !musicExtensions[ext] {
			return nil
		}

		if err := p.processFile(path); err != nil {
			fmt.Printf("エラー (%s): %v\n\n", path, err)
		}

		return nil
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("使用法:")
		fmt.Println("  音楽ファイル処理: go run main.go <音楽ファイルまたはディレクトリパス>")
		fmt.Println("  Spotify認証設定: SPOTIFY_CLIENT_ID と SPOTIFY_CLIENT_SECRET 環境変数を設定してください")
		os.Exit(1)
	}

	processor := NewMusicFileProcessor()

	// Spotify認証情報を取得
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("警告: Spotify認証情報が設定されていません")
		fmt.Println("SPOTIFY_CLIENT_ID と SPOTIFY_CLIENT_SECRET 環境変数を設定してください")
		fmt.Println("Spotifyの代わりに他の方法を使用しますか？ (y/N)")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			os.Exit(1)
		}

		fmt.Println("注意: 現在のバージョンではSpotify APIのみサポートしています")
		os.Exit(1)
	}

	// Spotifyトークンを取得
	fmt.Println("Spotify API認証中...")
	if err := processor.getSpotifyToken(clientID, clientSecret); err != nil {
		fmt.Printf("Spotify認証エラー: %v\n", err)
		os.Exit(1)
	}

	inputPath := os.Args[1]

	// ffmpegがインストールされているかチェック
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("エラー: ffmpegがインストールされていません")
		fmt.Println("ffmpegをインストールしてから再実行してください")
		os.Exit(1)
	}

	// ファイルまたはディレクトリの処理
	info, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("エラー: %v\n", err)
		os.Exit(1)
	}

	if info.IsDir() {
		fmt.Printf("ディレクトリを処理中: %s\n\n", inputPath)
		if err := processor.processDirectory(inputPath); err != nil {
			fmt.Printf("ディレクトリ処理エラー: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := processor.processFile(inputPath); err != nil {
			fmt.Printf("ファイル処理エラー: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("すべての処理が完了しました！")
}
