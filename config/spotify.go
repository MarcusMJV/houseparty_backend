package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type SpotifyAuth struct {
	AccessToken string
	IssuedAt    time.Time
	ExpiresIn   int
}

var spotifyAuth SpotifyAuth

func InitSpotify() {
	_, err := refreshAccessToken()
	if err != nil {
		log.Fatal("Failed to get Spotify token:", err)
	}
}

func GetAccessToken() (string, error) {

	if spotifyAuth.AccessToken == "" {
		return refreshAccessToken()
	}

	expiry := spotifyAuth.IssuedAt.Add(time.Duration(spotifyAuth.ExpiresIn) * time.Second)

	if time.Now().After(expiry) {
		return refreshAccessToken()
	}

	return spotifyAuth.AccessToken, nil
}

func GenerateSpotifyAuthRequest() (string, error) {
	redirectUrl := GetFrontendCallback()

	log.Println(redirectUrl)

	scope := "streaming user-read-email user-read-private user-modify-playback-state user-read-playback-state"

	state := generateRandomString(16)
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")

	if clientID == "" {
		return "", errors.New("missing spotify client id or client secret" + clientID)
	}

	data := url.Values{}
	data.Add("response_type", "code")
	data.Add("client_id", clientID)
	data.Add("scope", scope)
	data.Add("redirect_uri", redirectUrl)
	data.Add("state", state)

	authUrl := "https://accounts.spotify.com/authorize?" + data.Encode()

	return authUrl, nil
}

func generateRandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatal("Error generating random string: ", err)
	}
	return hex.EncodeToString(b)[:n]
}

func SetSpotifyToken(code string) (string, error) {

	log.Println("Spotify auth code:", code)

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURL := os.Getenv("FRONTEND_CALLBACK")

	if clientID == "" || clientSecret == "" {
		return "", errors.New("missing spotify client credentials")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURL)

	req, err := http.NewRequest(
		"POST",
		"https://accounts.spotify.com/api/token",
		strings.NewReader(data.Encode()),
	)

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	credentials := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+credentials)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", errors.New(string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}

	log.Println("Access Token:", tokenResp.AccessToken)
	log.Println("Refresh Token:", tokenResp.RefreshToken)

	// Save refresh token to env file
	if tokenResp.RefreshToken != "" {
		err := saveRefreshToken(tokenResp.RefreshToken)
		if err != nil {
			log.Println("Failed to store refresh token:", err)
		}
	}

	return tokenResp.AccessToken, nil
}

func saveRefreshToken(token string) error {

	envFile := ".env"

	content, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		if strings.HasPrefix(line, "SPOTIFY_REFRESH_TOKEN=") {
			lines[i] = "SPOTIFY_REFRESH_TOKEN=" + token
			found = true
		}
	}

	if !found {
		lines = append(lines, "SPOTIFY_REFRESH_TOKEN="+token)
	}

	newContent := strings.Join(lines, "\n")

	return os.WriteFile(envFile, []byte(newContent), 0644)
}

func refreshAccessToken() (string, error) {

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, _ := http.NewRequest(
		"POST",
		"https://accounts.spotify.com/api/token",
		strings.NewReader(data.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	json.NewDecoder(resp.Body).Decode(&tokenResp)

	spotifyAuth.AccessToken = tokenResp.AccessToken
	spotifyAuth.IssuedAt = time.Now()
	spotifyAuth.ExpiresIn = tokenResp.ExpiresIn

	return tokenResp.AccessToken, nil
}
