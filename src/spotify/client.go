package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client はSpotify APIクライアント
type Client struct {
	accessToken string
	httpClient  *http.Client
}

// NewClient は新しいSpotifyクライアントを作成
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetToken はSpotify Web APIのアクセストークンを取得
func (c *Client) GetToken(clientID, clientSecret string) error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
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

	c.accessToken = tokenResp.AccessToken
	return nil
}

// SearchArtwork はSpotify APIを使用してアートワークを検索（曲検索ベース）
func (c *Client) SearchArtwork(artist, title string) (string, error) {
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

	req.Header.Add("Authorization", "Bearer "+c.accessToken)
	fmt.Printf("Debug: 認証ヘッダー設定完了 (トークン長: %d文字)\n", len(c.accessToken))

	fmt.Printf("Debug: Spotify検索APIにリクエスト送信中...\n")
	resp, err := c.httpClient.Do(req)
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