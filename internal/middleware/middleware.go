// Package middleware содержит gin-middleware: JWT-аутентификацию (J-04/J-05)
// и проверку ролей (viewer/organizer/admin) для защищённых маршрутов.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/raxima/seatpicker/internal/httpx"
	"github.com/raxima/seatpicker/internal/jwtutil"
	"github.com/raxima/seatpicker/internal/model"
)

const (
	CtxUserID = "user_id"
	CtxRole   = "role"
)

func Auth(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			httpx.RespondError(c, model.ErrUnauthorized)
			c.Abort()
			return
		}

		token := strings.TrimPrefix(header, prefix)
		claims, err := jwtutil.ParseAccess(token, secret)
		if err != nil {
			httpx.RespondError(c, model.ErrUnauthorized)
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireRole(roles ...model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(CtxRole)
		if !exists {
			httpx.RespondError(c, model.ErrUnauthorized)
			c.Abort()
			return
		}

		role, _ := value.(model.Role)
		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}

		httpx.RespondError(c, model.ErrForbidden)
		c.Abort()
	}
}
