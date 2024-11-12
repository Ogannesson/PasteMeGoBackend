package token

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type UserInfo struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	TrustLevel int    `json:"trust_level"`
	Active     bool   `json:"active"`
	Silenced   bool   `json:"silenced"`
}

// verifyAccessToken 通过 OAuth 提供者的用户信息端点验证 access_token
func verifyAccessToken(accessToken string) (*UserInfo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://connect.linux.do/api/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid access token")
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func CheckAuthStatus(c *gin.Context) {
	accessToken, err := c.Cookie("access_token")
	if err != nil || accessToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "message": "User not logged in"})
		return
	}

	userInfo, err := verifyAccessToken(accessToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "authorized",
		"user_info": userInfo,
	})
}
