package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var AuthURL string

func init() {
	AuthURL = os.Getenv("ICANN_AUTH_URL")
	if AuthURL == "" {
		AuthURL = "https://account-api.icann.org/api/authenticate"
	}
}

func GetAccessToken(ctx context.Context, username, password string) (string, error) {
	input := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}
	body, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(AuthURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status_code=%d; check your username and/or password", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	output := &struct {
		AccessToken string `json:"accessToken"`
	}{}
	if err = json.Unmarshal(raw, output); err != nil {
		return "", err
	}
	return output.AccessToken, nil
}
