package metadata

import (
	"fmt"
	"os"

	"github.com/dhowden/tag"
)

// ExtractMetadata は音楽ファイルからメタデータを抽出
func ExtractMetadata(filePath string) (artist, album, title string, err error) {
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