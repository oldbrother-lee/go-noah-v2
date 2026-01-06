package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/gin-gonic/gin"
	"net/http"
	"go-noah/api"
	"go-noah/internal/model"
	"go-noah/pkg/jwt"
)

func AuthMiddleware(e *casbin.SyncedEnforcer) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从上下文获取用户信息（假设通过 JWT 或其他方式设置）
		v, exists := ctx.Get("claims")
		if !exists {
			api.HandleError(ctx, http.StatusUnauthorized, api.ErrUnauthorized, nil)
			ctx.Abort()
			return
		}
		uid := v.(*jwt.MyCustomClaims).UserId
		uidStr := convertor.ToString(uid)
		
		// 防呆设计：检查用户ID是否为1，或者用户是否有admin角色
		if uidStr == model.AdminUserID {
			// 用户ID为1，直接跳过权限检查
			ctx.Next()
			return
		}
		
		// 检查用户是否有admin角色
		roles, err := e.GetRolesForUser(uidStr)
		if err == nil {
			for _, role := range roles {
				if role == model.AdminRole {
					// 用户有admin角色，跳过API权限检查
					ctx.Next()
					return
				}
			}
		}

		// 获取请求的资源和操作
		sub := uidStr
		obj := model.ApiResourcePrefix + ctx.Request.URL.Path
		act := ctx.Request.Method

		// 检查权限
		allowed, err := e.Enforce(sub, obj, act)
		if err != nil {
			api.HandleError(ctx, http.StatusForbidden, api.ErrForbidden, nil)
			ctx.Abort()
			return
		}
		if !allowed {
			api.HandleError(ctx, http.StatusForbidden, api.ErrForbidden, nil)
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
