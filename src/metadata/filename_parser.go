package metadata

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ExtractTitleFromFilename はファイル名から曲名を抽出
func ExtractTitleFromFilename(filePath string) string {
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