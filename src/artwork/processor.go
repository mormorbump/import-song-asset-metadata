package artwork

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Processor はアートワーク処理を行う構造体
type Processor struct {
	httpClient *http.Client
}

// NewProcessor は新しいアートワークプロセッサーを作成
func NewProcessor() *Processor {
	return &Processor{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DownloadImage は指定されたURLから画像をダウンロード
func (p *Processor) DownloadImage(imageURL, outputPath string) error {
	resp, err := p.httpClient.Get(imageURL)
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

// GetAudioFormat は音楽ファイルのフォーマットを取得
func (p *Processor) GetAudioFormat(musicFile string) (string, error) {
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

// HasExistingArtwork は音楽ファイルに既存のアートワークがあるかチェック
func (p *Processor) HasExistingArtwork(musicFile string) (bool, error) {
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

// EmbedArtwork はffmpegを使用してアートワークを埋め込み
func (p *Processor) EmbedArtwork(musicFile, artworkFile, outputFile string) error {
	// 入力ファイルのフォーマットを取得
	format, err := p.GetAudioFormat(musicFile)
	if err != nil {
		return fmt.Errorf("フォーマット取得エラー: %w", err)
	}

	fmt.Printf("    検出されたフォーマット: %s\n", format)

	// フォーマット別の処理
	switch format {
	case "mp3":
		return EmbedArtworkMP3(musicFile, artworkFile, outputFile)
	case "mp4":
		return EmbedArtworkMP4(musicFile, artworkFile, outputFile)
	case "flac":
		return EmbedArtworkFLAC(musicFile, artworkFile, outputFile)
	default:
		return EmbedArtworkGeneric(musicFile, artworkFile, outputFile, format)
	}
}

// EmbedArtworkForceReplace は既存アートワークを強制置換
func (p *Processor) EmbedArtworkForceReplace(musicFile, artworkFile, outputFile string) error {
	// 入力ファイルのフォーマットを取得
	format, err := p.GetAudioFormat(musicFile)
	if err != nil {
		return fmt.Errorf("フォーマット取得エラー: %w", err)
	}

	fmt.Printf("    検出されたフォーマット: %s\n", format)

	// フォーマット別の処理
	switch format {
	case "mp3":
		return EmbedArtworkForceReplaceMP3(musicFile, artworkFile, outputFile)
	case "mp4":
		return EmbedArtworkForceReplaceMP4(musicFile, artworkFile, outputFile)
	case "flac":
		return EmbedArtworkForceReplaceFLAC(musicFile, artworkFile, outputFile)
	default:
		return EmbedArtworkForceReplaceGeneric(musicFile, artworkFile, outputFile, format)
	}
}