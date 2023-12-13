package tesla

import (
	"encoding/json"
	"github.com/L480/tesla-smart-sentry/internal/logger"
	"github.com/L480/tesla-smart-sentry/internal/request"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"
)

func RefreshToken(c Config) {
	tokenTimer := time.NewTimer(0)
	refreshInterval := 6 * time.Hour
	retryInterval := 5 * time.Minute
	var latestRefreshToken string
	type response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	for {
		<-tokenTimer.C
		logger.Info("Refreshing access token...")
		file, err := os.ReadFile(c.RefreshTokenFile)
		if err != nil {
			latestRefreshToken = c.RefreshToken
		} else {
			latestRefreshToken = string(file)
		}
		form := url.Values{}
		form.Add("grant_type", "refresh_token")
		form.Add("client_id", c.ClientId)
		form.Add("refresh_token", latestRefreshToken)
		tokenEndpoint := request.Endpoint{
			Url:                "https://auth.tesla.com/oauth2/v3/token",
			Method:             "POST",
			Headers:            map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			Body:               form.Encode(),
			InsecureSkipVerify: false,
		}
		resp, err := request.Connect(tokenEndpoint)
		if err != nil {
			logger.Error("Couldn't connect to token endpoint: %s", err)
			tokenTimer.Reset(retryInterval)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("Couldn't retrieve response body: %s", err)
			tokenTimer.Reset(retryInterval)
			continue
		}
		if resp.StatusCode == 200 {
			logger.Info("Refresh successful. Next refresh scheduled in %s hours.", strconv.FormatFloat(refreshInterval.Hours(), 'g', 2, 64))
			tokenTimer.Reset(refreshInterval)
		} else {
			logger.Error("Refresh failed: %s.", string(body))
			tokenTimer.Reset(retryInterval)
			continue
		}
		var jsonData response
		json.Unmarshal(body, &jsonData)
		accessTokenFile, err := os.Create(c.AccessTokenFile)
		if err != nil {
			logger.Error("Couldn't save access token: %s", err)
			tokenTimer.Reset(retryInterval)
			continue
		}
		accessTokenFile.WriteString(jsonData.AccessToken)
		accessTokenFile.Close()
		refreshTokenFile, err := os.Create(c.RefreshTokenFile)
		if err != nil {
			logger.Error("Couldn't save refresh token: %s", err)
			tokenTimer.Reset(retryInterval)
			continue
		}
		refreshTokenFile.WriteString(jsonData.RefreshToken)
		refreshTokenFile.Close()
	}
}
