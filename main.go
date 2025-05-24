package main

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// SpotifySearchResponse はSpotify検索APIのレスポンス構造体
type SpotifySearchResponse struct {
	Tracks struct {
		Items []struct {
			Name  string `json:"name"`
			Album struct {
				Name   string `json:"name"`
				Images []struct {
					URL    string `json:"url"`
					Height int    `json:"height"`
					Width  int    `json:"width"`
				} `json:"images"`
			} `json:"album"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

// Config はアプリケーションの設定を管理
type Config struct {
	ForceOverwrite bool
}

// parseArgs はコマンドライン引数を解析
func parseArgs() (inputPath string, config *Config, err error) {
	config = &Config{}

	if len(os.Args) < 2 {
		return "", nil, fmt.Errorf("insufficient arguments")
	}

	args := os.Args[1:]
	var inputFound bool

	for _, arg := range args {
		switch arg {
		case "--force", "-f":
			config.ForceOverwrite = true
		case "--help", "-h":
			return "", nil, fmt.Errorf("help requested")
		default:
			if !inputFound && !strings.HasPrefix(arg, "-") {
				inputPath = arg
				inputFound = true
			}
		}
	}

	if !inputFound {
		return "", nil, fmt.Errorf("no input path specified")
	}

	return inputPath, config, nil
}

// min は2つの整数の最小値を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractTitleFromFilename はファイル名から曲名を抽出
func (p *MusicFileProcessor) extractTitleFromFilename(filePath string) string {
	// ファイル名のみを取得（パスを除く）
	filename := filepath.Base(filePath)

	// 拡張子を除去
	titleWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 先頭の番号パターンを除去
	// パターン例: "01 ", "01. ", "1 ", "1. ", "01_", "01-"
	patterns := []string{
		`^\d{1,3}[\s\.\-_]+`, // 1-3桁の数字 + 区切り文字
		`^\d{1,3}`,           // 1-3桁の数字のみ（区切り文字なし）
	}

	cleanTitle := titleWithoutExt
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(cleanTitle) {
			cleanTitle = re.ReplaceAllString(cleanTitle, "")
			break // 最初にマッチしたパターンのみ適用
		}
	}

	// 前後の空白を削除
	cleanTitle = strings.TrimSpace(cleanTitle)

	// アンダースコアをスペースに変換（オプション）
	cleanTitle = strings.ReplaceAll(cleanTitle, "_", " ")

	fmt.Printf("    ファイル名から抽出: '%s' -> '%s'\n", titleWithoutExt, cleanTitle)

	return cleanTitle
}

// MusicFileProcessor は音楽ファイルの処理を行う構造体
type MusicFileProcessor struct {
	spotifyToken string
	client       *http.Client
	config       *Config
}

// NewMusicFileProcessor は新しいプロセッサーを作成
func NewMusicFileProcessor(config *Config) *MusicFileProcessor {
	return &MusicFileProcessor{
		client: &http.Client{Timeout: 30 * time.Second},
		config: config,
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

// searchArtwork はSpotify APIを使用してアートワークを検索（曲検索ベース）
func (p *MusicFileProcessor) searchArtwork(artist, title string) (string, error) {
	fmt.Printf("Debug: アートワーク検索開始\n")
	fmt.Printf("Debug: アーティスト: '%s'\n", artist)
	fmt.Printf("Debug: 曲名: '%s'\n", title)

	// 曲名とアーティスト名で検索
	query := fmt.Sprintf("track:%s artist:%s", title, artist)
	encodedQuery := url.QueryEscape(query)

	fmt.Printf("Debug: 検索クエリ: '%s'\n", query)
	fmt.Printf("Debug: エンコード済みクエリ: '%s'\n", encodedQuery)

	searchURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=1", encodedQuery)
	fmt.Printf("Debug: 検索URL: %s\n", searchURL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		fmt.Printf("Debug: リクエスト作成エラー: %v\n", err)
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+p.spotifyToken)
	fmt.Printf("Debug: 認証ヘッダー設定完了 (トークン長: %d文字)\n", len(p.spotifyToken))

	fmt.Printf("Debug: Spotify検索APIにリクエスト送信中...\n")
	resp, err := p.client.Do(req)
	if err != nil {
		fmt.Printf("Debug: HTTPリクエストエラー: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	fmt.Printf("Debug: 検索レスポンスステータス: %d\n", resp.StatusCode)

	// レスポンスボディを読み取り
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Debug: レスポンス読み取りエラー: %v\n", err)
		return "", err
	}

	fmt.Printf("Debug: 検索レスポンスボディ: %s\n", string(body))

	var searchResp SpotifySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		fmt.Printf("Debug: JSON解析エラー: %v\n", err)
		return "", err
	}

	fmt.Printf("Debug: 検索結果楽曲数: %d\n", len(searchResp.Tracks.Items))

	if len(searchResp.Tracks.Items) == 0 {
		fmt.Printf("Debug: 楽曲が見つかりませんでした\n")
		return "", fmt.Errorf("アートワークが見つかりませんでした")
	}

	firstTrack := searchResp.Tracks.Items[0]
	fmt.Printf("Debug: 見つかった楽曲: '%s'\n", firstTrack.Name)
	fmt.Printf("Debug: 楽曲のアーティスト: %v\n", firstTrack.Artists)
	fmt.Printf("Debug: アルバム名: '%s'\n", firstTrack.Album.Name)
	fmt.Printf("Debug: 画像数: %d\n", len(firstTrack.Album.Images))

	if len(firstTrack.Album.Images) == 0 {
		fmt.Printf("Debug: アルバムに画像がありません\n")
		return "", fmt.Errorf("アートワークが見つかりませんでした")
	}

	// 最高解像度の画像を選択
	bestImage := firstTrack.Album.Images[0]
	for i, img := range firstTrack.Album.Images {
		fmt.Printf("Debug: 画像%d - URL: %s, サイズ: %dx%d\n", i, img.URL, img.Width, img.Height)
		if img.Height > bestImage.Height {
			bestImage = img
		}
	}

	fmt.Printf("Debug: 選択された画像: %s (%dx%d)\n", bestImage.URL, bestImage.Width, bestImage.Height)

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

// getAudioFormat は音楽ファイルのフォーマットを取得
func (p *MusicFileProcessor) getAudioFormat(musicFile string) (string, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		musicFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var probeResult struct {
		Format struct {
			FormatName string `json:"format_name"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return "", err
	}

	// フォーマット名から適切な出力フォーマットを決定
	formatName := probeResult.Format.FormatName

	// 複数のフォーマットが含まれている場合（例: "mp3,mp2,mp1"）、最初のものを使用
	if strings.Contains(formatName, ",") {
		formatName = strings.Split(formatName, ",")[0]
	}

	// ffmpegで使用する出力フォーマット名にマッピング
	switch formatName {
	case "mp3":
		return "mp3", nil
	case "mp4":
		return "mp4", nil
	case "mov,mp4,m4a,3gp,3g2,mj2": // M4Aファイルの場合
		return "mp4", nil
	case "flac":
		return "flac", nil
	case "wav":
		return "wav", nil
	default:
		// デフォルトはファイル拡張子から推定
		ext := strings.ToLower(filepath.Ext(musicFile))
		switch ext {
		case ".mp3":
			return "mp3", nil
		case ".m4a", ".mp4":
			return "mp4", nil
		case ".flac":
			return "flac", nil
		case ".wav":
			return "wav", nil
		default:
			return "mp3", nil // フォールバック
		}
	}
}

// embedArtworkForceReplace は既存アートワークを強制置換
func (p *MusicFileProcessor) embedArtworkForceReplace(musicFile, artworkFile, outputFile string) error {
	// 入力ファイルのフォーマットを取得
	format, err := p.getAudioFormat(musicFile)
	if err != nil {
		return fmt.Errorf("フォーマット取得エラー: %w", err)
	}

	fmt.Printf("    検出されたフォーマット: %s\n", format)

	// フォーマット別の処理
	switch format {
	case "mp3":
		return p.embedArtworkForceReplaceMP3(musicFile, artworkFile, outputFile)
	case "mp4":
		return p.embedArtworkForceReplaceMP4(musicFile, artworkFile, outputFile)
	case "flac":
		return p.embedArtworkForceReplaceFLAC(musicFile, artworkFile, outputFile)
	default:
		return p.embedArtworkForceReplaceGeneric(musicFile, artworkFile, outputFile, format)
	}
}

// embedArtworkForceReplaceMP3 はMP3ファイルの既存アートワークを強制置換
func (p *MusicFileProcessor) embedArtworkForceReplaceMP3(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a", // 音声ストリームのみ
		"-map", "1:0", // 新しい画像
		"-c:a", "copy", // 音声はコピー
		"-c:v", "mjpeg", // 画像はmjpegでエンコード
		"-id3v2_version", "3",
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", `comment=Cover (front)`,
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("MP3強制置換エラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// embedArtworkForceReplaceMP4 はMP4/M4Aファイルの既存アートワークを強制置換
func (p *MusicFileProcessor) embedArtworkForceReplaceMP4(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a", // 音声ストリームのみ（既存画像を除外）
		"-map", "1:0", // 新しい画像
		"-c:a", "copy", // 音声はコピー
		"-c:v", "png", // M4Aの場合、PNGまたはJPEGが推奨
		"-disposition:v:0", "attached_pic",
		"-tag:v:0", "hvc1", // M4A用の画像タグ
		"-f", "mp4",
		"-movflags", "+faststart", // M4A最適化
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// PNGで失敗した場合、JPEGで再試行
		fmt.Printf("    PNG強制置換失敗、JPEGで再試行中...\n")
		cmd = exec.Command("ffmpeg",
			"-i", musicFile,
			"-i", artworkFile,
			"-map", "0:a",
			"-map", "1:0",
			"-c:a", "copy",
			"-c:v", "mjpeg", // JPEG形式
			"-disposition:v:0", "attached_pic",
			"-f", "mp4",
			"-movflags", "+faststart",
			"-y",
			outputFile,
		)

		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("M4A強制置換エラー: %w\n出力: %s", err, string(output))
		}
	}

	return nil
}

// embedArtworkForceReplaceFLAC はFLACファイルの既存アートワークを強制置換
func (p *MusicFileProcessor) embedArtworkForceReplaceFLAC(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a",
		"-map", "1:0",
		"-c:a", "copy",
		"-c:v", "copy",
		"-disposition:v:0", "attached_pic",
		"-f", "flac",
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FLAC強制置換エラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// embedArtworkForceReplaceGeneric は汎用の既存アートワーク強制置換
func (p *MusicFileProcessor) embedArtworkForceReplaceGeneric(musicFile, artworkFile, outputFile, format string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a",
		"-map", "1:0",
		"-c:a", "copy",
		"-c:v", "copy",
		"-disposition:v:0", "attached_pic",
		"-f", format,
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("汎用強制置換エラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// embedArtwork はffmpegを使用してアートワークを埋め込み
func (p *MusicFileProcessor) embedArtwork(musicFile, artworkFile, outputFile string) error {
	// 入力ファイルのフォーマットを取得
	format, err := p.getAudioFormat(musicFile)
	if err != nil {
		return fmt.Errorf("フォーマット取得エラー: %w", err)
	}

	fmt.Printf("    検出されたフォーマット: %s\n", format)

	// フォーマット別の処理
	switch format {
	case "mp3":
		return p.embedArtworkMP3(musicFile, artworkFile, outputFile)
	case "mp4":
		return p.embedArtworkMP4(musicFile, artworkFile, outputFile)
	case "flac":
		return p.embedArtworkFLAC(musicFile, artworkFile, outputFile)
	default:
		return p.embedArtworkGeneric(musicFile, artworkFile, outputFile, format)
	}
}

// embedArtworkMP3 はMP3ファイル専用の画像埋め込み
func (p *MusicFileProcessor) embedArtworkMP3(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:0", // 音声ストリーム
		"-map", "1:0", // 画像ストリーム
		"-c:a", "copy", // 音声はコピー
		"-c:v", "mjpeg", // 画像はmjpegとしてエンコード
		"-id3v2_version", "3",
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", `comment=Cover (front)`,
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("MP3埋め込みエラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// embedArtworkMP4 はMP4/M4Aファイル専用の画像埋め込み（音声ファイルのみ）
func (p *MusicFileProcessor) embedArtworkMP4(musicFile, artworkFile, outputFile string) error {
	// まず標準的な方法を試行
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a", // 音声ストリーム
		"-map", "1:0", // 画像ストリーム
		"-c:a", "copy", // 音声はコピー
		"-c:v", "copy", // 画像もコピー（再エンコードなし）
		"-disposition:v:0", "attached_pic",
		"-metadata:s:v:0", "title=Album cover",
		"-metadata:s:v:0", `comment=Cover (front)`,
		"-f", "mp4",
		"-movflags", "+faststart",
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 標準方法で失敗した場合、AtomicParsleyを使用する方法に切り替え
		fmt.Printf("    標準埋め込み失敗、代替方法で再試行中...\n")

		// 一時的に音声のみのファイルを作成
		tempAudioPath := outputFile + ".temp.m4a"
		cmd = exec.Command("ffmpeg",
			"-i", musicFile,
			"-map", "0:a",
			"-c:a", "copy",
			"-f", "mp4",
			"-y",
			tempAudioPath,
		)

		if _, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("音声抽出エラー: %w", err)
		}
		defer os.Remove(tempAudioPath)

		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("M4A画像埋め込みエラー（全手法失敗）: %w\n出力: %s", err, string(output))
		}
	}

	return nil
}

// embedArtworkFLAC はFLACファイル専用の画像埋め込み
func (p *MusicFileProcessor) embedArtworkFLAC(musicFile, artworkFile, outputFile string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a",
		"-map", "1:0",
		"-c:a", "copy",
		"-c:v", "copy",
		"-disposition:v:0", "attached_pic",
		"-f", "flac",
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FLAC埋め込みエラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// embedArtworkGeneric は汎用の画像埋め込み
func (p *MusicFileProcessor) embedArtworkGeneric(musicFile, artworkFile, outputFile, format string) error {
	cmd := exec.Command("ffmpeg",
		"-i", musicFile,
		"-i", artworkFile,
		"-map", "0:a",
		"-map", "1:0",
		"-c:a", "copy",
		"-c:v", "copy",
		"-disposition:v:0", "attached_pic",
		"-f", format,
		"-y",
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("汎用埋め込みエラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// processFile は単一の音楽ファイルを処理
func (p *MusicFileProcessor) processFile(filePath string) error {
	fmt.Printf("処理中: %s\n", filePath)

	// 元ファイルのバックアップを作成
	backupPath := filePath + ".backup"
	if err := p.createBackup(filePath, backupPath); err != nil {
		return fmt.Errorf("バックアップ作成エラー: %w", err)
	}
	defer func() {
		// 処理完了後、バックアップを削除（成功時のみ）
		if _, err := os.Stat(backupPath); err == nil {
			os.Remove(backupPath)
		}
	}()

	// 既存のアートワークをチェック
	hasArtwork, err := p.hasExistingArtwork(filePath)
	if err != nil {
		fmt.Printf("  警告: アートワーク確認に失敗しました (%v)。処理を続行します。\n", err)
	} else if hasArtwork && !p.config.ForceOverwrite {
		fmt.Printf("  既存のアートワークが検出されました。スキップします。\n")
		fmt.Printf("  強制上書きする場合は --force または -f オプションを使用してください。\n\n")
		return nil
	} else if hasArtwork && p.config.ForceOverwrite {
		fmt.Printf("  既存のアートワークが検出されましたが、強制上書きモードで処理を続行します。\n")
	}

	// メタデータを抽出
	artist, album, title, err := p.extractMetadata(filePath)
	if err != nil {
		return fmt.Errorf("メタデータ抽出エラー: %w", err)
	}

	fmt.Printf("  アーティスト: %s\n", artist)
	fmt.Printf("  アルバム: %s\n", album)
	fmt.Printf("  タイトル: %s\n", title)

	// 検索に使用する情報を決定
	searchArtist := artist
	searchTitle := title

	// アーティスト情報が不足している場合のフォールバック
	if artist == "" {
		searchArtist = "Unknown Artist"
		fmt.Printf("  警告: アーティスト情報がありません。'Unknown Artist' で検索します。\n")
	}

	// タイトル情報が不足している場合、ファイル名から抽出
	if title == "" {
		searchTitle = p.extractTitleFromFilename(filePath)
		if searchTitle == "" {
			fmt.Printf("  警告: タイトル情報とファイル名から曲名を抽出できませんでした。スキップします。\n\n")
			return nil
		}
		fmt.Printf("  ファイル名から抽出した曲名で検索: %s\n", searchTitle)
	}

	// 最低限の情報（アーティストまたはタイトル）があるかチェック
	if searchTitle == "" {
		fmt.Printf("  警告: 検索に必要な情報が不足しています。スキップします。\n\n")
		return nil
	}

	// アートワークを検索
	fmt.Println("  アートワークを検索中...")
	artworkURL, err := p.searchArtwork(searchArtist, searchTitle)
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
	if hasArtwork && p.config.ForceOverwrite {
		fmt.Println("  既存アートワークを置き換え中...")
		if err := p.embedArtworkForceReplace(filePath, tempImagePath, tempOutputPath); err != nil {
			// 失敗した場合、バックアップから復元
			p.restoreFromBackup(backupPath, filePath)
			return fmt.Errorf("アートワーク埋め込みエラー: %w", err)
		}
	} else {
		fmt.Println("  アートワークを埋め込み中...")
		if err := p.embedArtwork(filePath, tempImagePath, tempOutputPath); err != nil {
			// 失敗した場合、バックアップから復元
			p.restoreFromBackup(backupPath, filePath)
			return fmt.Errorf("アートワーク埋め込みエラー: %w", err)
		}
	}

	// 一時ファイルの整合性をチェック
	if err := p.validateAudioFile(tempOutputPath); err != nil {
		os.Remove(tempOutputPath) // 破損ファイルを削除
		p.restoreFromBackup(backupPath, filePath)
		return fmt.Errorf("出力ファイル検証エラー: %w", err)
	}

	// 元ファイルを一時ファイルで置き換え
	if err := os.Rename(tempOutputPath, filePath); err != nil {
		os.Remove(tempOutputPath) // クリーンアップ
		p.restoreFromBackup(backupPath, filePath)
		return fmt.Errorf("ファイル置き換えエラー: %w", err)
	}

	fmt.Printf("  完了: %s\n\n", filePath)
	return nil
}

// createBackup はファイルのバックアップを作成
func (p *MusicFileProcessor) createBackup(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// restoreFromBackup はバックアップからファイルを復元
func (p *MusicFileProcessor) restoreFromBackup(backupPath, originalPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("バックアップファイルが存在しません")
	}

	fmt.Printf("  エラー検出: バックアップから復元中...\n")
	return os.Rename(backupPath, originalPath)
}

// validateAudioFile は音声ファイルの整合性をチェック
func (p *MusicFileProcessor) validateAudioFile(filePath string) error {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("ファイル検証失敗: %w", err)
	}

	// 出力が空でないことを確認
	if len(strings.TrimSpace(string(output))) == 0 {
		return fmt.Errorf("ファイルが破損しています（duration取得不可）")
	}

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

// hasExistingArtwork は音楽ファイルに既存のアートワークがあるかチェック
func (p *MusicFileProcessor) hasExistingArtwork(musicFile string) (bool, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		musicFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var probeResult struct {
		Streams []struct {
			CodecType   string `json:"codec_type"`
			CodecName   string `json:"codec_name"`
			Disposition struct {
				AttachedPic int `json:"attached_pic"`
			} `json:"disposition"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return false, err
	}

	// ビデオストリームでattached_picがあるかチェック
	for _, stream := range probeResult.Streams {
		if stream.CodecType == "video" && stream.Disposition.AttachedPic == 1 {
			return true, nil
		}
	}

	return false, nil
}

func main() {
	inputPath, config, err := parseArgs()
	if err != nil {
		if err.Error() == "help requested" {
			fmt.Println("使用法:")
			fmt.Println("  音楽ファイル処理: go run main.go [オプション] <音楽ファイルまたはディレクトリパス>")
			fmt.Println("")
			fmt.Println("オプション:")
			fmt.Println("  -f, --force    既存のアートワークを強制的に上書きする")
			fmt.Println("  -h, --help     このヘルプを表示する")
			fmt.Println("")
			fmt.Println("環境変数:")
			fmt.Println("  SPOTIFY_CLIENT_ID     Spotify API Client ID")
			fmt.Println("  SPOTIFY_CLIENT_SECRET Spotify API Client Secret")
			fmt.Println("")
			fmt.Println("例:")
			fmt.Println("  go run main.go music.mp3                    # 単一ファイルを処理")
			fmt.Println("  go run main.go /path/to/music/directory     # ディレクトリを処理")
			fmt.Println("  go run main.go -f music.mp3                 # 既存アートワークを強制上書き")
			os.Exit(0)
		}
		fmt.Println("エラー:", err)
		fmt.Println("使用法: go run main.go [オプション] <音楽ファイルまたはディレクトリパス>")
		fmt.Println("詳細は --help を参照してください")
		os.Exit(1)
	}

	processor := NewMusicFileProcessor(config)

	// .envファイルを読み込み
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Spotify認証情報を取得
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		fmt.Println("警告: Spotify認証情報が設定されていません")
		fmt.Println("SPOTIFY_CLIENT_ID と SPOTIFY_CLIENT_SECRET 環境変数を設定してください")
		os.Exit(1)
	}

	// Spotifyトークンを取得
	fmt.Println("Spotify API認証中...")
	if err := processor.getSpotifyToken(clientID, clientSecret); err != nil {
		fmt.Printf("Spotify認証エラー: %v\n", err)
		os.Exit(1)
	}

	// ffmpegがインストールされているかチェック
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("エラー: ffmpegがインストールされていません")
		fmt.Println("ffmpegをインストールしてから再実行してください")
		os.Exit(1)
	}

	// ffprobeがインストールされているかチェック
	if _, err := exec.LookPath("ffprobe"); err != nil {
		fmt.Println("エラー: ffprobeがインストールされていません")
		fmt.Println("ffprobe（ffmpegパッケージに含まれる）をインストールしてから再実行してください")
		os.Exit(1)
	}

	// 強制上書きモードの表示
	if config.ForceOverwrite {
		fmt.Println("強制上書きモード: 既存のアートワークを置き換えます")
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
