package paste

import (
	"github.com/PasteUs/PasteMeGoBackend/handler/session"
	"github.com/PasteUs/PasteMeGoBackend/logging"
	model "github.com/PasteUs/PasteMeGoBackend/model/paste"
	"github.com/PasteUs/PasteMeGoBackend/model/user"
	"github.com/PasteUs/PasteMeGoBackend/util"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func pasteValidator(lang string, content string) error {
	if content == "" {
		return ErrEmptyContent // 内容为空，返回错误信息 "empty content"
	}
	if lang == "" {
		return ErrEmptyLang // 语言类型为空，返回错误信息 "empty lang"
	}
	return nil
}

func expireValidator(expireType string, expiration uint64) error {
	if expireType == "" {
		return ErrEmptyExpireType
	}
	if expiration <= 0 {
		return ErrZeroExpiration
	}

	if expireType == model.EnumTime {
		if expiration > model.OneMonth {
			return ErrExpirationGreaterThanMonth
		}
	} else if expireType == model.EnumCount {
		if expiration > model.MaxCount {
			return ErrExpirationGreaterThanMaxCount
		}
	} else {
		return ErrInvalidExpireType
	}
	return nil
}

func Create(context *gin.Context) {
	u := session.AuthMiddleware.IdentityHandler(context).(*user.User)
	namespace := u.Username
	logging.Info("create paste", context, zap.String("namespace", namespace))

	body := struct {
		*model.AbstractPaste
		SelfDestruct bool   `json:"self_destruct"`
		ExpireType   string `json:"expire_type"`
		Expiration   uint64 `json:"expiration"`
	}{
		AbstractPaste: &model.AbstractPaste{
			ClientIP:  context.ClientIP(),
			Namespace: namespace,
		},
	}

	if err := context.ShouldBindJSON(&body); err != nil {
		logging.Warn("bind body failed", context, zap.String("err", err.Error()))
		context.JSON(http.StatusOK, gin.H{
			"status":  http.StatusBadRequest,
			"message": "wrong param type",
		})
		return
	}

	if err := func() error {
		if e := pasteValidator(body.Lang, body.Content); e != nil {
			return e
		}
		if body.SelfDestruct {
			if e := expireValidator(body.ExpireType, body.Expiration); e != nil {
				return e
			}
		}
		return nil
	}(); err != nil {
		logging.Info("param validate failed", zap.String("err", err.Error()))
		context.JSON(http.StatusOK, gin.H{
			"status":  http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	if body.AbstractPaste.Password != "" {
		body.AbstractPaste.Password = util.String2md5(body.AbstractPaste.Password)
	}

	var paste model.IPaste

	if body.SelfDestruct {
		paste = &model.Temporary{
			Key:           model.Generator(),
			AbstractPaste: body.AbstractPaste,
			ExpireType:    body.ExpireType,
			Expiration:    body.Expiration,
		}
	} else {
		paste = &model.Permanent{AbstractPaste: body.AbstractPaste}
	}

	if err := paste.Save(); err != nil {
		logging.Warn("save failed", context, zap.String("err", err.Error()))
		context.JSON(http.StatusOK, gin.H{
			"status":  http.StatusInternalServerError,
			"message": ErrSaveFailed,
		})
		return
	}

	context.JSON(http.StatusCreated, gin.H{
		"status":    http.StatusCreated,
		"key":       paste.GetKey(),
		"namespace": paste.GetNamespace(),
	})
}

func Get(context *gin.Context) {
	namespace, key := context.Param("namespace"), context.Param("key")

	var (
		table string
		err   error
		paste model.IPaste
	)

	if table, err = util.ValidChecker(key); err != nil {
		context.JSON(http.StatusOK, gin.H{
			"status":  http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	abstractPaste := model.AbstractPaste{Namespace: namespace}

	if table == "temporary" {
		paste = &model.Temporary{Key: key, AbstractPaste: &abstractPaste}
	} else {
		paste = &model.Permanent{Key: util.String2uint(key), AbstractPaste: &abstractPaste}
	}

	if err = paste.Get(context.DefaultQuery("password", "")); err != nil {
		var (
			status  int
			message string
		)

		switch err {
		case gorm.ErrRecordNotFound:
			status = http.StatusNotFound
			message = err.Error()
		case model.ErrWrongPassword:
			status = http.StatusForbidden
			message = err.Error()
		default:
			logging.Error("query from db failed", context, zap.String("err", err.Error()))
			status = http.StatusInternalServerError
			message = ErrQueryDBFailed.Error()
		}

		context.JSON(http.StatusOK, gin.H{
			"status":  status,
			"message": message,
		})

		return
	}

	if strings.Contains(context.GetHeader("Accept"), "json") {
		context.JSON(http.StatusOK, gin.H{
			"status":  http.StatusOK,
			"lang":    paste.GetLang(),
			"content": paste.GetContent(),
		})
	} else {
		context.String(http.StatusOK, paste.GetContent())
	}
}
