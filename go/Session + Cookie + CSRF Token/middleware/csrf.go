package middleware

import (
	"net/http"
	"session-csrf/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// unsafeMethods是需要进行CSRF验证的HTTP方法集合。
var unsafeMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// IssueCsrfToken 生成（或重用）一个 CSRF 令牌并将其存储在
// session。以JSON格式返回令牌。
//
// 相当于 Express 的 issueCsrfToken 方法：
// 如果 (!req.session.csrfToken) { req.session.csrfToken = generateCsrfToken(); }
// res.json({ csrfToken: req.session.csrfToken });；
func IssueCsrfToken(c *gin.Context) {
	session := sessions.Default(c)
	// 类型断言：session.Get 返回的是 interface{}，检查 token 是否存在
	existing, _ := session.Get("csrfToken").(string)
	if existing == "" {
		token, err := utils.GenerateCsrfToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "csrfToken generation failed"})
			return
		}
		session.Set("csrfToken", token)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session save failed"})
			return
		}
		existing = token
	}
	c.JSON(http.StatusOK, gin.H{"csrfToken": existing})
}

// CsrfGuard 返回一个 Gin 中间件，该中间件会阻止不安全的 HTTP 方法，除非
// 一个有效的X-CSRF-Token头部与会话中存储的token相匹配。
//
// 相当于Express框架中的csrfGuard功能：
// - GET/HEAD/OPTIONS 请求直接通过
// - POST/PUT/PATCH/DELETE操作需要X-CSRF-Token头
// - token 与 timingSafeEqual 的比较
func CsrfGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !unsafeMethods[c.Request.Method] {
			c.Next()
			return
		}
		session := sessions.Default(c)
		sessionToken, _ := session.Get("csrfToken").(string)
		clientToken := c.GetHeader("X-CSRF-Token")
		if sessionToken == "" || clientToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			return
		}
		if !utils.SafeEqual(clientToken, sessionToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token invalid"})
			return
		}
		c.Next()
	}
}

// RegenerateCsrfToken 用一个新的CSRF令牌覆盖会话中的CSRF令牌。
// 登录后调用以防止会话固定攻击。返回新的令牌。
//
// 相当于Express的后登录模式：
// 请求会话中的csrf令牌 = 生成csrf令牌();；
func RegenerateCsrfToken(c *gin.Context) (string, error) {
	session := sessions.Default(c)
	token, err := utils.GenerateCsrfToken()
	if err != nil {
		return "", err
	}
	session.Set("csrfToken", token)
	if err := session.Save(); err != nil {
		return "", err
	}
	return token, nil
}
