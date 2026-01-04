# 按 gin-vue-admin 方式注册路由完成

## 重构内容

按照 gin-vue-admin 的风格，完成了路由注册方式的重构。

## 主要变更

### 1. Handler 改为全局变量

#### `internal/handler/admin.go`
```go
// 之前：需要创建实例
func NewAdminHandler(handler *Handler) *AdminHandler {
    return &AdminHandler{
        Handler: handler,
    }
}

// 现在：全局变量（gin-vue-admin 风格）
var AdminHandlerApp = &AdminHandler{}

// 初始化 Handler 的 base（在应用启动时调用一次）
func InitHandler(logger *log.Logger) {
    baseHandler := NewHandler(logger)
    AdminHandlerApp.Handler = baseHandler
    UserHandlerApp.Handler = baseHandler
}
```

#### `internal/handler/user.go`
```go
// 之前：需要创建实例
func NewUserHandler(handler *Handler) *UserHandler {
    return &UserHandler{
        Handler: handler,
    }
}

// 现在：全局变量（gin-vue-admin 风格）
var UserHandlerApp = &UserHandler{}
```

### 2. 创建 router 包

#### `internal/router/router.go`（新建）
```go
package router

import (
    "go-noah/internal/handler"
    "go-noah/internal/middleware"
    "go-noah/pkg/jwt"
    "go-noah/pkg/log"
    "github.com/casbin/casbin/v2"
    "github.com/gin-gonic/gin"
)

// InitRouter 初始化路由（gin-vue-admin 风格）
func InitRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
    // 初始化各个模块的路由
    InitAdminRouter(r, jwt, e, logger)
    InitUserRouter(r, jwt, e, logger)
}

// InitAdminRouter 初始化管理员相关路由
func InitAdminRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
    v1 := r.Group("/v1")
    {
        noAuthRouter := v1.Group("/")
        {
            noAuthRouter.POST("/login", handler.AdminHandlerApp.Login)  // 直接使用全局 Handler
        }
        
        strictAuthRouter := v1.Group("/").Use(middleware.StrictAuth(jwt, logger), middleware.AuthMiddleware(e))
        {
            strictAuthRouter.GET("/menus", handler.AdminHandlerApp.GetMenus)
            // ... 其他路由
        }
    }
}

// InitUserRouter 初始化用户相关路由
func InitUserRouter(r *gin.Engine, jwt *jwt.JWT, e *casbin.SyncedEnforcer, logger *log.Logger) {
    v1 := r.Group("/v1")
    {
        strictAuthRouter := v1.Group("/").Use(middleware.StrictAuth(jwt, logger), middleware.AuthMiddleware(e))
        {
            strictAuthRouter.GET("/users", handler.UserHandlerApp.GetUsers)  // 直接使用全局 Handler
        }
    }
}
```

### 3. 简化 server/http.go

#### `internal/server/http.go`
```go
// 之前：需要传入 Handler 参数
func NewHTTPServer(
    logger *log.Logger,
    conf *viper.Viper,
    jwt *jwt.JWT,
    e *casbin.SyncedEnforcer,
    adminHandler *handler.AdminHandler,  // 需要传入
    userHandler *handler.UserHandler,   // 需要传入
) *http.Server {
    // ... 在函数内部注册路由
    v1 := s.Group("/v1")
    {
        noAuthRouter.POST("/login", adminHandler.Login)
        // ...
    }
}

// 现在：不需要传入 Handler 参数
func NewHTTPServer(
    logger *log.Logger,
    conf *viper.Viper,
    jwt *jwt.JWT,
    e *casbin.SyncedEnforcer,
) *http.Server {
    // ... 中间件设置
    
    // 使用 router 包注册路由（gin-vue-admin 风格）
    router.InitRouter(s.Engine, jwt, e, logger)
    
    return s
}
```

### 4. 简化 noah.go

#### `pkg/noah/noah.go`
```go
// 之前：需要创建 Handler 实例并传入
func NewServerApp(...) {
    // ...
    h := handler.NewHandler(logger)
    adminHandler := handler.NewAdminHandler(h)
    userHandler := handler.NewUserHandler(h)
    
    httpServer := server.NewHTTPServer(logger, conf, global.JWT, global.Enforcer, adminHandler, userHandler)
}

// 现在：只初始化 Handler 的 base
func NewServerApp(...) {
    // ...
    // Handler 初始化为全局变量（gin-vue-admin 风格）
    handler.InitHandler(logger)
    
    // 不再需要传入 Handler 参数（router 中直接使用全局 Handler）
    httpServer := server.NewHTTPServer(logger, conf, global.JWT, global.Enforcer)
}
```

## 对比 gin-vue-admin

### gin-vue-admin 的方式

```go
// handler/user.go
type UserHandler struct{}

var UserHandlerApp = new(UserHandler)  // 全局变量

// router/user.go
func InitUserRouter(Router *gin.RouterGroup) {
    UserRouter := Router.Group("user")
    {
        UserRouter.POST("register", handler.UserHandlerApp.Register)  // 直接使用全局 Handler
    }
}

// initialize/router.go
func Routers() *gin.Engine {
    Router := gin.Default()
    router.InitUserRouter(Router)  // 初始化路由
    return Router
}
```

### go-noah 现在的实现

```go
// handler/admin.go
type AdminHandler struct {
    *Handler
}

var AdminHandlerApp = &AdminHandler{}  // 全局变量

// router/router.go
func InitAdminRouter(r *gin.Engine, ...) {
    v1 := r.Group("/v1")
    {
        v1.POST("/login", handler.AdminHandlerApp.Login)  // 直接使用全局 Handler
    }
}

// server/http.go
func NewHTTPServer(...) *http.Server {
    // ...
    router.InitRouter(s.Engine, jwt, e, logger)  // 初始化路由
    return s
}
```

## 优势

1. **更符合 gin-vue-admin 风格**：Handler 使用全局变量，路由在 router 包中注册
2. **代码更简洁**：不需要在应用初始化时创建 Handler 实例并传入
3. **路由管理更清晰**：所有路由集中在 `router` 包中，便于维护
4. **扩展更方便**：新增路由只需在 `router` 包中添加，不需要修改 `server/http.go`

## 文件变更清单

- ✅ `internal/handler/admin.go` - Handler 改为全局变量，添加 `InitHandler` 函数
- ✅ `internal/handler/user.go` - Handler 改为全局变量
- ✅ `internal/router/router.go` - 新建，路由注册逻辑
- ✅ `internal/server/http.go` - 简化，移除 Handler 参数，调用 router 注册路由
- ✅ `pkg/noah/noah.go` - 简化，只初始化 Handler 的 base

## 完成状态

✅ 所有重构已完成，代码已通过 linter 检查，无语法错误。

