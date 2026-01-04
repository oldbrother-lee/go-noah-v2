package router

import (
	"go-noah/internal/handler"
	"go-noah/internal/middleware"
	"go-noah/pkg/jwt"
	"go-noah/pkg/log"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
	// 初始化各个模块的路由
	InitAdminRouter(r, jwt, e, logger)
	InitUserRouter(r, jwt, e, logger)
}

// InitAdminRouter 初始化管理员相关路由
func InitAdminRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
	v1 := r.Group("/v1")
	{
		// No route group has permission
		noAuthRouter := v1.Group("/")
		{
			noAuthRouter.POST("/login", handler.AdminHandlerApp.Login)
		}

		// Strict permission routing group
		strictAuthRouter := v1.Group("/").Use(middleware.StrictAuth(jwt, logger), middleware.AuthMiddleware(e))
		{
			strictAuthRouter.GET("/menus", handler.AdminHandlerApp.GetMenus)
			strictAuthRouter.GET("/admin/menus", handler.AdminHandlerApp.GetAdminMenus)
			strictAuthRouter.POST("/admin/menu", handler.AdminHandlerApp.MenuCreate)
			strictAuthRouter.PUT("/admin/menu", handler.AdminHandlerApp.MenuUpdate)
			strictAuthRouter.DELETE("/admin/menu", handler.AdminHandlerApp.MenuDelete)

			strictAuthRouter.GET("/admin/users", handler.AdminHandlerApp.GetAdminUsers)
			strictAuthRouter.GET("/admin/user", handler.AdminHandlerApp.GetAdminUser)
			strictAuthRouter.PUT("/admin/user", handler.AdminHandlerApp.AdminUserUpdate)
			strictAuthRouter.POST("/admin/user", handler.AdminHandlerApp.AdminUserCreate)
			strictAuthRouter.DELETE("/admin/user", handler.AdminHandlerApp.AdminUserDelete)
			strictAuthRouter.GET("/admin/user/permissions", handler.AdminHandlerApp.GetUserPermissions)
			strictAuthRouter.GET("/admin/role/permissions", handler.AdminHandlerApp.GetRolePermissions)
			strictAuthRouter.PUT("/admin/role/permission", handler.AdminHandlerApp.UpdateRolePermission)
			strictAuthRouter.GET("/admin/roles", handler.AdminHandlerApp.GetRoles)
			strictAuthRouter.POST("/admin/role", handler.AdminHandlerApp.RoleCreate)
			strictAuthRouter.PUT("/admin/role", handler.AdminHandlerApp.RoleUpdate)
			strictAuthRouter.DELETE("/admin/role", handler.AdminHandlerApp.RoleDelete)

			strictAuthRouter.GET("/admin/apis", handler.AdminHandlerApp.GetApis)
			strictAuthRouter.POST("/admin/api", handler.AdminHandlerApp.ApiCreate)
			strictAuthRouter.PUT("/admin/api", handler.AdminHandlerApp.ApiUpdate)
			strictAuthRouter.DELETE("/admin/api", handler.AdminHandlerApp.ApiDelete)
		}
	}
}

// InitUserRouter 初始化用户相关路由
func InitUserRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
	v1 := r.Group("/v1")
	{
		// Strict permission routing group
		strictAuthRouter := v1.Group("/").Use(middleware.StrictAuth(jwt, logger), middleware.AuthMiddleware(e))
		{
			strictAuthRouter.GET("/users", handler.UserHandlerApp.GetUsers)
		}
	}
}
