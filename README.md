# 音楽ファイル自動アートワーク埋め込みツール

Spotify APIを使用して音楽ファイルに自動でアルバムアートワークを埋め込むGoアプリケーションです。

## 機能

- 音楽ファイルのメタデータ（アーティスト・アルバム情報）を自動抽出
- Spotify APIを使用したアートワーク画像の自動検索
- 高品質な画像の自動ダウンロード
- ffmpegを使用したアートワークの音楽ファイルへの埋め込み
- ディレクトリ内の複数ファイルの一括処理
- メタデータ不足ファイルのスキップ機能

## 対応フォーマット

- MP3 (.mp3)
- M4A (.m4a)
- FLAC (.flac)
- WAV (.wav)

## 必要な環境

### 1. ffmpegのインストール

#### macOS
```bash
# Homebrewを使用
brew install ffmpeg
```

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install ffmpeg
```

#### CentOS/RHEL/Rocky Linux
```bash
# EPELリポジトリを有効化
sudo dnf install epel-release
sudo dnf install ffmpeg
```

#### Windows
1. [公式サイト](https://ffmpeg.org/download.html)からWindows版をダウンロード
2. 解凍してPATHに追加
3. または以下のパッケージマネージャーを使用：
```powershell
# Chocolateyを使用
choco install ffmpeg

# Scoopを使用
scoop install ffmpeg
```

#### Arch Linux
```bash
sudo pacman -S ffmpeg
```

### 2. Go言語の環境
Go 1.21以上が必要です。

### 3. Spotify API認証情報
[Spotify Developer Console](https://developer.spotify.com/dashboard/)でアプリケーションを作成し、Client IDとClient Secretを取得してください。

## セットアップ

### 1. リポジトリのクローンと依存関係のインストール
```bash
git clone <repository-url>
cd music-artwork-embedder
go mod tidy
```

### 2. Spotify API認証情報の設定
```bash
export SPOTIFY_CLIENT_ID="your_spotify_client_id"
export SPOTIFY_CLIENT_SECRET="your_spotify_client_secret"
```

Windows（PowerShell）の場合：
```powershell
$env:SPOTIFY_CLIENT_ID="your_spotify_client_id"
$env:SPOTIFY_CLIENT_SECRET="your_spotify_client_secret"
```

## 使用方法

### 単一ファイルの処理
```bash
go run main.go /path/to/music/file.mp3
```

### ディレクトリ全体の一括処理
```bash
go run main.go /path/to/music/directory
```

### 実行可能ファイルとしてビルド
```bash
go build -o music-artwork-embedder
./music-artwork-embedder /path/to/music/file.mp3
```

## Spotify Developer Console設定手順

1. [Spotify Developer Console](https://developer.spotify.com/dashboard/)にアクセス
2. Spotifyアカウントでログイン
3. 「Create App」をクリック
4. アプリ情報を入力：
    - App name: 任意の名前
    - App description: 任意の説明
    - Redirect URI: `http://localhost` (使用しませんが必須)
    - API/SDKs: Web API にチェック
5. 作成後、「Settings」から Client ID と Client Secret を取得

## 動作の流れ

1. 音楽ファイルまたはディレクトリを指定して実行
2. 各音楽ファイルからメタデータ（アーティスト・アルバム情報）を抽出
3. Spotify APIを使用してアートワークを検索
4. 最高品質の画像をダウンロード
5. ffmpegを使用して画像を音楽ファイルに埋め込み
6. 元のファイルを上書き保存

## エラーハンドリング

### スキップされるファイル
- メタデータ（アーティスト・アルバム情報）が不足している音楽ファイル
- Spotify APIでアートワークが見つからない音楽ファイル
- 対応していないファイル形式

### 警告メッセージ
```
警告: アーティストまたはアルバム情報が不足しています。スキップします。
警告: アートワーク検索に失敗しました (アートワークが見つかりませんでした)。スキップします。
```

## 注意事項

- **ファイルの上書き**: 処理により元のファイルが上書きされます。事前にバックアップを取ることを推奨します
- **API制限**: Spotify APIには使用制限があります。大量のファイルを処理する際は注意してください
- **メタデータ要件**: アーティスト名またはアルバム名のいずれかが必要です
- **ネットワーク**: インターネット接続が必要です

## トラブルシューティング

### ffmpegが見つからない
```
エラー: ffmpegがインストールされていません
```
→ 上記のffmpegインストール手順を参照してください

### Spotify認証エラー
```
Spotify認証エラー: ...
```
→ SPOTIFY_CLIENT_IDとSPOTIFY_CLIENT_SECRETが正しく設定されているか確認してください

### メタデータが読み取れない
```
メタデータを読み取れませんでした: ...
```
→ ファイルが破損しているか、対応していない形式の可能性があります

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。

## 依存関係

- [github.com/dhowden/tag](https://github.com/dhowden/tag) - 音楽ファイルメタデータ読み取り
- Go標準ライブラリ
- ffmpeg（外部依存）