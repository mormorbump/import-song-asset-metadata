package main

import (
	"fmt"
	"os"
	"os/exec"

	"music-artwork-embedder/src/args"
	"music-artwork-embedder/src/config"
	"music-artwork-embedder/src/orchestrator"
)

func main() {
	// コマンドライン引数を解析
	inputPath, argsConfig, err := args.ParseArgs()
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

	// 設定を初期化
	cfg := config.NewConfig(argsConfig.ForceOverwrite)

	// 環境変数を読み込み
	if err := cfg.LoadEnv(); err != nil {
		fmt.Printf("環境変数読み込みエラー: %v\n", err)
		os.Exit(1)
	}

	// Spotify認証情報を検証
	if err := cfg.ValidateSpotifyCredentials(); err != nil {
		fmt.Printf("警告: %v\n", err)
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

	// オーケストレーターを作成・初期化
	orch := orchestrator.NewOrchestrator(cfg)
	if err := orch.Initialize(); err != nil {
		fmt.Printf("初期化エラー: %v\n", err)
		os.Exit(1)
	}

	// 強制上書きモードの表示
	if cfg.ForceOverwrite {
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
		if err := orch.ProcessDirectory(inputPath); err != nil {
			fmt.Printf("ディレクトリ処理エラー: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := orch.ProcessFile(inputPath); err != nil {
			fmt.Printf("ファイル処理エラー: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("すべての処理が完了しました！")
}
