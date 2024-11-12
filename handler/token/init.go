package token

import (
	"encoding/json"
	"github.com/PasteUs/PasteMeGoBackend/common/config"
	"github.com/PasteUs/PasteMeGoBackend/common/logging"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"time"
)

var (
	IdentityKey    = "username"   // 用户身份标识键
	AuthMiddleware *JWTMiddleware // JWT 认证中间件
)

// OAuthUser OAuth 用户信息结构
type OAuthUser struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Active     bool   `json:"active"`
	TrustLevel int    `json:"trust_level"`
	Silenced   bool   `json:"silenced"`
}

// 初始化 OAuth 配置
func initOAuth() {
	oauthConfig = &oauth2.Config{
		ClientID:     "jO9crndsdHaLMM5gpfVSmlOuKPTNwNLp",
		ClientSecret: "eWgXfX32ojeEugix1jrZllMjHY87UsMA",
		RedirectURL:  "https://paste.linuxdoi.ng/oauth/callback", // 更新回调 URL
		Scopes:       []string{"user"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://connect.linux.do/oauth2/authorize",
			TokenURL: "https://connect.linux.do/oauth2/token",
		},
	}
}

// authenticator 通过 OAuth 验证用户身份
func authenticator(c *gin.Context) (interface{}, error) {
	token, err := c.Cookie("oauth_token")
	if err != nil || token == "" {
		return nil, jwt.ErrFailedAuthentication
	}

	req, err := http.NewRequest("GET", "https://connect.linux.do/api/user", nil)
	if err != nil {
		return nil, jwt.ErrFailedAuthentication
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, jwt.ErrFailedAuthentication
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logging.Error("close body failed", zap.Error(err))
		}
	}(resp.Body)

	var user OAuthUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, jwt.ErrFailedAuthentication
	}

	if !user.Active || user.Silenced {
		return nil, jwt.ErrFailedAuthentication
	}

	return user.Username, nil
}

// payloadFunc 用于生成 JWT 负载
func payloadFunc(data interface{}) jwt.MapClaims {
	if username, ok := data.(string); ok {
		return jwt.MapClaims{
			IdentityKey: username,
		}
	}
	return jwt.MapClaims{}
}

// init 初始化 JWT 和 OAuth
func init() {
	initOAuth()
	var err error
	AuthMiddleware = &JWTMiddleware{
		&jwt.GinJWTMiddleware{
			Realm:         "pasteme",
			Key:           []byte(config.Config.Secret),
			Timeout:       time.Hour,
			MaxRefresh:    time.Hour,
			IdentityKey:   IdentityKey,
			Authenticator: authenticator,
			PayloadFunc:   payloadFunc,
			TokenLookup:   "cookie: token",
			TokenHeadName: "PasteMe",
		},
	}

	if err = AuthMiddleware.MiddlewareInit(); err != nil {
		logging.Panic("jwt middleware init failed", zap.Error(err))
	}
}
