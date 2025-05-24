package fileutils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateBackup はファイルのバックアップを作成
func CreateBackup(src, dst string) error {
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

// RestoreFromBackup はバックアップからファイルを復元
func RestoreFromBackup(backupPath, originalPath string) error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("バックアップファイルが存在しません")
	}

	fmt.Printf("  エラー検出: バックアップから復元中...\n")
	return os.Rename(backupPath, originalPath)
}

// ValidateAudioFile は音声ファイルの整合性をチェック
func ValidateAudioFile(filePath string) error {
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

// ProcessDirectory はディレクトリ内の音楽ファイルを再帰的に処理
func ProcessDirectory(dirPath string, processFunc func(string) error) error {
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

		if err := processFunc(path); err != nil {
			fmt.Printf("エラー (%s): %v\n\n", path, err)
		}

		return nil
	})
}