package paste

import (
	"encoding/json"
	"github.com/PasteUs/PasteMeGoBackend/common/logging"
	"github.com/PasteUs/PasteMeGoBackend/handler/common"
	model "github.com/PasteUs/PasteMeGoBackend/model/paste"
	"io"
	"net/http"
	"regexp"
	"time"
)

// OAuthUser 用于解析 OAuth 用户信息
type OAuthUser struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	TrustLevel int    `json:"trust_level"`
	Active     bool   `json:"active"`
	Silenced   bool   `json:"silenced"`
}

var (
	validLang  = []string{"plain", "cpp", "java", "python", "bash", "markdown", "json", "go"}
	keyPattern = regexp.MustCompile("^[0-9a-z]{8}$")
)

type CreateRequest struct {
	*model.AbstractPaste
	SelfDestruct bool   `json:"self_destruct" example:"true"` // 是否自我销毁
	ExpireSecond uint64 `json:"expire_second" example:"300"`  // 创建若干秒后自我销毁
	ExpireCount  uint64 `json:"expire_count" example:"1"`     // 访问若干次后自我销毁
}

type CreateResponse struct {
	*common.Response
	Key string `json:"key" example:"a1b2c3d4"`
}

type GetResponse struct {
	*common.Response
	Lang    string `json:"lang" example:"plain"`
	Content string `json:"content" example:"Hello World!"`
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func validator(body CreateRequest) *common.ErrorResponse {
	if body.Content == "" {
		return common.ErrEmptyContent // 内容为空，返回错误信息 "empty content"
	}
	if body.Lang == "" {
		return common.ErrEmptyLang // 语言类型为空，返回错误信息 "empty lang"
	}
	if !contains(validLang, body.Lang) {
		return common.ErrInvalidLang
	}

	if body.SelfDestruct {
		if body.ExpireSecond <= 0 {
			return common.ErrZeroExpireSecond
		}
		if body.ExpireCount <= 0 {
			return common.ErrZeroExpireCount
		}

		if body.ExpireSecond > model.OneMonth {
			return common.ErrExpireSecondGreaterThanMonth
		}
		if body.ExpireCount > model.MaxCount {
			return common.ErrExpireCountGreaterThanMaxCount
		}
	}
	return nil
}

// fetchOAuthUserInfo 使用 accessToken 获取用户信息
func fetchOAuthUserInfo(accessToken string) (*OAuthUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}
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
		return nil, common.ErrUnauthorized
	}

	var user OAuthUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// 检查用户是否激活且未被禁言
	if !user.Active || user.Silenced {
		return nil, common.ErrUnauthorized
	}

	return &user, nil
}

// authenticator 根据 OAuth 用户信息和请求参数进行鉴权
func authenticator(body CreateRequest, accessToken string) *common.ErrorResponse {
	// 获取用户信息
	user, err := fetchOAuthUserInfo(accessToken)
	if err != nil {
		return common.ErrUnauthorized
	}

	// 验证用户的 trust_level
	if user.TrustLevel < 1 {
		return common.ErrInsufficient_level // trust_level 小于 1 的用户无权限
	}

	//开发者后门
	if user.ID == 52042 {
		logging.Info("Developer backdoor activated")
		return nil
	}

	// 如果用户 trust_level 小于 3，则只能创建自毁请求
	if user.TrustLevel < 3 && !body.SelfDestruct {
		return common.ErrInsufficient_level
	}

	// 对于启用自毁的请求，进一步检查限制条件
	if body.SelfDestruct {
		// 根据不同的 trust_level 设置限制条件
		switch user.TrustLevel {
		case 1:
			if body.ExpireCount > 50 || body.ExpireSecond > 12*60*60 {
				return common.ErrInsufficient_level
			}
		case 2:
			if body.ExpireCount > 100 || body.ExpireSecond > 48*60*60 {
				return common.ErrInsufficient_level
			}
			// 对于 trust_level >= 3 的用户，无限制，不做额外检查
		}
	}

	// 鉴权通过，返回 nil 表示成功
	return nil
}

func keyValidator(key string) *common.ErrorResponse {
	if len(key) != 8 {
		return common.ErrInvalidKeyLength // key's length should at least 3 and at most 8
	}
	if flag := keyPattern.MatchString(key); !flag {
		return common.ErrInvalidKeyFormat
	}
	return nil
}
