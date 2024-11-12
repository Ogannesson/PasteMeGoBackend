package paste

import (
	"errors"
	"github.com/PasteUs/PasteMeGoBackend/common/logging"
	"github.com/PasteUs/PasteMeGoBackend/handler/common"
	model "github.com/PasteUs/PasteMeGoBackend/model/paste"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// Create 创建一贴
// @Summary 创建永久存储或者是自我销毁的一贴
// @Description 只有在登陆的状态下才能创建永久的一贴
// @Tags Paste
// @Accept json
// @Produce json
// @Param Authorization header string false "登陆的 Token"
// @Param data body CreateRequest true "请求数据"
// @Success 201 {object} CreateResponse
// @Failure default {object} common.ErrorResponse
// @Router /paste/ [post]
func Create(context *gin.Context) {
	// 从 Cookie 中获取 access_token
	accessToken, err := context.Cookie("access_token")
	if err != nil || accessToken == "" {
		logging.Info("missing access token in cookie, unauthorized")
		common.ErrUnauthorized.Abort(context)
		return
	}

	// 验证 access_token（调用 authenticator 函数或其他验证逻辑）
	var requestBody CreateRequest
	if err := context.ShouldBindJSON(&requestBody); err != nil {
		logging.Warn("bind body failed", zap.Error(err))
		common.ErrWrongParamType.Abort(context)
		return
	}

	// 鉴权逻辑，可以使用 authenticator 函数或者直接在此处验证
	if err := authenticator(requestBody, accessToken); err != nil {
		logging.Info("unauthorized request")
		err.Abort(context)
		return
	}

	// 处理创建 Paste 的逻辑
	var paste model.IPaste
	if requestBody.SelfDestruct {
		paste = &model.Temporary{
			AbstractPaste: requestBody.AbstractPaste,
			ExpireSecond:  requestBody.ExpireSecond,
			ExpireCount:   requestBody.ExpireCount,
		}
	} else {
		paste = &model.Permanent{AbstractPaste: requestBody.AbstractPaste}
	}

	if err := paste.Save(); err != nil {
		logging.Error("save failed", zap.Error(err))
		common.ErrSaveFailed.Abort(context)
		return
	}

	// 返回成功响应
	common.JSON(context, CreateResponse{
		Response: &common.Response{Code: http.StatusCreated},
		Key:      paste.GetKey(),
	})
}

// Get godoc
// @Summary 读取一贴
// @Description 如果不指定 Accept: application/json 的话，默认会返回 text/plain 格式的 content
// @Tags Paste
// @Accept json
// @Produce json
// @Param Accept header string false "响应格式" default("text/plain")
// @Param key path string true "索引"
// @Success 201 {object} GetResponse
// @Failure default {object} common.ErrorResponse
// @Router /paste/{key} [get]
func Get(context *gin.Context) {
	key := strings.ToLower(context.Param("key"))

	var paste model.IPaste

	if err := keyValidator(key); err != nil {
		err.Abort(context)
		return
	}

	abstractPaste := model.AbstractPaste{Key: key}

	if []rune(key)[0] == '0' {
		paste = &model.Temporary{AbstractPaste: &abstractPaste}
	} else {
		paste = &model.Permanent{AbstractPaste: &abstractPaste}
	}

	if err := paste.Get(context.DefaultQuery("password", "")); err != nil {
		var errorResponse *common.ErrorResponse
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			errorResponse = common.ErrRecordNotFound
		case errors.Is(err, common.ErrWrongPassword):
			errors.As(err, &errorResponse)
		default:
			logging.Error("query from db failed", context, zap.Error(err))
			errorResponse = common.ErrQueryDBFailed
		}

		errorResponse.Abort(context)
		return
	}

	if strings.Contains(context.GetHeader("Accept"), "json") {
		common.JSON(context, GetResponse{
			Response: &common.Response{
				Code: http.StatusOK,
			},
			Lang:    paste.GetLang(),
			Content: paste.GetContent(),
		})
	} else {
		context.String(http.StatusOK, paste.GetContent())
	}
}
