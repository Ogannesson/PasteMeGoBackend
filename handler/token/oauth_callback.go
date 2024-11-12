package token

import (
	"github.com/PasteUs/PasteMeGoBackend/common/logging"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"net/http"
)

// OAuth2 配置
var oauthConfig = &oauth2.Config{
	ClientID:     "jO9crndsdHaLMM5gpfVSmlOuKPTNwNLp",
	ClientSecret: "eWgXfX32ojeEugix1jrZllMjHY87UsMA",
	RedirectURL:  "https://paste.linuxdoi.ng/oauth/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://connect.linux.do/oauth2/authorize",
		TokenURL: "https://connect.linux.do/oauth2/token",
	},
}

// OAuthCallback OAuth 回调处理函数
func OAuthCallback(c *gin.Context) {
	// 从查询参数中获取授权码
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	// 使用授权码换取访问令牌
	token, err := oauthConfig.Exchange(c, code)
	if err != nil {
		logging.Error("failed to exchange token", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token"})
		return
	}

	// 获取访问令牌
	accessToken := token.AccessToken

	// 将访问令牌存储到 Cookie 中，设置 HttpOnly 和 Secure 属性以提高安全性
	c.SetCookie("access_token", accessToken, 3600, "/", "paste.linuxdoi.ng", true, true)

	// 重定向到应用首页，表示登录成功
	c.Redirect(http.StatusFound, "/")
}
