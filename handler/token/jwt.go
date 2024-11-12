package token

import (
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"net/http"
)

// JWTMiddleware 扩展了 GinJWTMiddleware，增加了自定义的 unauthorized 处理
type JWTMiddleware struct {
	*jwt.GinJWTMiddleware
}

// MiddlewareFunc 返回认证中间件函数
// disableAboard 控制是否在认证失败时继续处理请求
func (mw *JWTMiddleware) MiddlewareFunc(disableAboard bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		mw.middlewareImpl(c, disableAboard)
	}
}

// unauthorized 处理未授权请求
func (mw *JWTMiddleware) unauthorized(disableAboard bool, c *gin.Context, code int, message string) {
	if disableAboard {
		c.Set(mw.IdentityKey, nil)
		c.Next()
	} else {
		c.Header("WWW-Authenticate", "JWT realm="+mw.Realm)
		if !mw.DisabledAbort {
			c.Abort()
		}
		mw.Unauthorized(c, code, message) // 调用默认的未授权处理函数
	}
}

// middlewareImpl 核心认证逻辑
func (mw *JWTMiddleware) middlewareImpl(c *gin.Context, disableAboard bool) {
	// 从 JWT 中提取声明 (claims)
	claims, err := mw.GetClaimsFromJWT(c)
	if err != nil {
		mw.unauthorized(disableAboard, c, http.StatusUnauthorized, mw.HTTPStatusMessageFunc(err, c))
		return
	}

	// 检查 "exp" 字段是否存在且格式正确
	exp, ok := claims["exp"].(float64)
	if !ok {
		mw.unauthorized(disableAboard, c, http.StatusBadRequest, mw.HTTPStatusMessageFunc(jwt.ErrWrongFormatOfExp, c))
		return
	}

	// 检查 Token 是否过期
	if int64(exp) < mw.TimeFunc().Unix() {
		mw.unauthorized(disableAboard, c, http.StatusUnauthorized, mw.HTTPStatusMessageFunc(jwt.ErrExpiredToken, c))
		return
	}

	// 设置有效的 JWT 声明
	c.Set("JWT_PAYLOAD", claims)
	identity := mw.IdentityHandler(c)

	// 将身份标识存储到上下文
	if identity != nil {
		c.Set(mw.IdentityKey, identity)
	}

	// 检查用户权限
	if !mw.Authorizator(identity, c) {
		mw.unauthorized(disableAboard, c, http.StatusForbidden, mw.HTTPStatusMessageFunc(jwt.ErrForbidden, c))
		return
	}

	// 继续处理请求
	c.Next()
}
