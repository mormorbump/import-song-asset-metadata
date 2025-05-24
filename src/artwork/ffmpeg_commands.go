package artwork

import (
	"fmt"
	"os/exec"
)

// EmbedArtworkMP3 はMP3ファイル専用の画像埋め込み
func EmbedArtworkMP3(musicFile, artworkFile, outputFile string) error {
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

// EmbedArtworkMP4 はMP4/M4Aファイル専用の画像埋め込み（音声ファイルのみ）
func EmbedArtworkMP4(musicFile, artworkFile, outputFile string) error {
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
		return fmt.Errorf("M4A画像埋め込みエラー: %w\n出力: %s", err, string(output))
	}

	return nil
}

// EmbedArtworkFLAC はFLACファイル専用の画像埋め込み
func EmbedArtworkFLAC(musicFile, artworkFile, outputFile string) error {
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

// EmbedArtworkGeneric は汎用の画像埋め込み
func EmbedArtworkGeneric(musicFile, artworkFile, outputFile, format string) error {
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

// EmbedArtworkForceReplaceMP3 はMP3ファイルの既存アートワークを強制置換
func EmbedArtworkForceReplaceMP3(musicFile, artworkFile, outputFile string) error {
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

// EmbedArtworkForceReplaceMP4 はMP4/M4Aファイルの既存アートワークを強制置換
func EmbedArtworkForceReplaceMP4(musicFile, artworkFile, outputFile string) error {
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

// EmbedArtworkForceReplaceFLAC はFLACファイルの既存アートワークを強制置換
func EmbedArtworkForceReplaceFLAC(musicFile, artworkFile, outputFile string) error {
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

// EmbedArtworkForceReplaceGeneric は汎用の既存アートワーク強制置換
func EmbedArtworkForceReplaceGeneric(musicFile, artworkFile, outputFile, format string) error {
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
