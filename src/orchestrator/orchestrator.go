package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"

	"music-artwork-embedder/src/artwork"
	"music-artwork-embedder/src/config"
	"music-artwork-embedder/src/fileutils"
	"music-artwork-embedder/src/metadata"
	"music-artwork-embedder/src/spotify"
)

// Orchestrator は各モジュールを協調させて処理を行う
type Orchestrator struct {
	config           *config.Config
	spotifyClient    *spotify.Client
	artworkProcessor *artwork.Processor
}

// NewOrchestrator は新しいオーケストレーターを作成
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	return &Orchestrator{
		config:           cfg,
		spotifyClient:    spotify.NewClient(),
		artworkProcessor: artwork.NewProcessor(),
	}
}

// Initialize はSpotifyクライアントを初期化
func (o *Orchestrator) Initialize() error {
	// Spotifyトークンを取得
	fmt.Println("Spotify API認証中...")
	if err := o.spotifyClient.GetToken(o.config.SpotifyClientID, o.config.SpotifyClientSecret); err != nil {
		return fmt.Errorf("Spotify認証エラー: %w", err)
	}
	return nil
}

// ProcessFile は単一の音楽ファイルを処理
func (o *Orchestrator) ProcessFile(filePath string) error {
	fmt.Printf("処理中: %s\n", filePath)

	// 元ファイルのバックアップを作成
	backupPath := filePath + ".backup"
	if err := fileutils.CreateBackup(filePath, backupPath); err != nil {
		return fmt.Errorf("バックアップ作成エラー: %w", err)
	}
	defer func() {
		// 処理完了後、バックアップを削除（成功時のみ）
		if _, err := os.Stat(backupPath); err == nil {
			os.Remove(backupPath)
		}
	}()

	// 既存のアートワークをチェック
	hasArtwork, err := o.artworkProcessor.HasExistingArtwork(filePath)
	if err != nil {
		fmt.Printf("  警告: アートワーク確認に失敗しました (%v)。処理を続行します。\n", err)
	} else if hasArtwork && !o.config.ForceOverwrite {
		fmt.Printf("  既存のアートワークが検出されました。スキップします。\n")
		fmt.Printf("  強制上書きする場合は --force または -f オプションを使用してください。\n\n")
		return nil
	} else if hasArtwork && o.config.ForceOverwrite {
		fmt.Printf("  既存のアートワークが検出されましたが、強制上書きモードで処理を続行します。\n")
	}

	// メタデータを抽出
	artist, album, title, err := metadata.ExtractMetadata(filePath)
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
		searchTitle = metadata.ExtractTitleFromFilename(filePath)
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
	artworkURL, err := o.spotifyClient.SearchArtwork(searchArtist, searchTitle)
	if err != nil {
		fmt.Printf("  警告: アートワーク検索に失敗しました (%v)。スキップします。\n\n", err)
		return nil
	}

	// 一時ファイルパスを生成
	tempImagePath := filepath.Join(os.TempDir(), "temp_artwork.jpg")
	defer os.Remove(tempImagePath)

	// 画像をダウンロード
	fmt.Println("  アートワークをダウンロード中...")
	if err := o.artworkProcessor.DownloadImage(artworkURL, tempImagePath); err != nil {
		return fmt.Errorf("画像ダウンロードエラー: %w", err)
	}

	// 一時出力ファイルパスを生成（元ファイルを上書きするため）
	tempOutputPath := filePath + ".tmp"

	// アートワークを埋め込み
	if hasArtwork && o.config.ForceOverwrite {
		fmt.Println("  既存アートワークを置き換え中...")
		if err := o.artworkProcessor.EmbedArtworkForceReplace(filePath, tempImagePath, tempOutputPath); err != nil {
			// 失敗した場合、バックアップから復元
			fileutils.RestoreFromBackup(backupPath, filePath)
			return fmt.Errorf("アートワーク埋め込みエラー: %w", err)
		}
	} else {
		fmt.Println("  アートワークを埋め込み中...")
		if err := o.artworkProcessor.EmbedArtwork(filePath, tempImagePath, tempOutputPath); err != nil {
			// 失敗した場合、バックアップから復元
			fileutils.RestoreFromBackup(backupPath, filePath)
			return fmt.Errorf("アートワーク埋め込みエラー: %w", err)
		}
	}

	// 一時ファイルの整合性をチェック
	if err := fileutils.ValidateAudioFile(tempOutputPath); err != nil {
		os.Remove(tempOutputPath) // 破損ファイルを削除
		fileutils.RestoreFromBackup(backupPath, filePath)
		return fmt.Errorf("出力ファイル検証エラー: %w", err)
	}

	// 元ファイルを一時ファイルで置き換え
	if err := os.Rename(tempOutputPath, filePath); err != nil {
		os.Remove(tempOutputPath) // クリーンアップ
		fileutils.RestoreFromBackup(backupPath, filePath)
		return fmt.Errorf("ファイル置き換えエラー: %w", err)
	}

	fmt.Printf("  完了: %s\n\n", filePath)
	return nil
}

// ProcessDirectory はディレクトリ内の音楽ファイルを再帰的に処理
func (o *Orchestrator) ProcessDirectory(dirPath string) error {
	return fileutils.ProcessDirectory(dirPath, o.ProcessFile)
}
