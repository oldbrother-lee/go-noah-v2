package server

import (
	"context"
	"go-noah/api"
	"go-noah/docs"
	"go-noah/internal/middleware"
	"go-noah/internal/router"
	"go-noah/internal/service"
	"go-noah/pkg/jwt"
	"go-noah/pkg/log"
	"go-noah/pkg/server/http"
	nethttp "net/http"
	"os"

	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewHTTPServer(
	logger *log.Logger,
	conf *viper.Viper,
	jwt *jwt.JWT,
	e *casbin.SyncedEnforcer,
) *http.Server {
	gin.SetMode(gin.DebugMode)
	s := http.NewServer(
		gin.Default(),
		logger,
		http.WithServerHost(conf.GetString("http.host")),
		http.WithServerPort(conf.GetInt("http.port")),
	)
	// 设置前端静态资源（使用外部 web 目录）
	s.Use(static.Serve("/", static.LocalFile("../web/dist", true)))
	s.NoRoute(func(c *gin.Context) {
		indexPageData, err := os.ReadFile("../web/dist/index.html")
		if err != nil {
			c.String(nethttp.StatusNotFound, "404 page not found")
			return
		}
		c.Data(nethttp.StatusOK, "text/html; charset=utf-8", indexPageData)
	})
	// swagger doc
	docs.SwaggerInfo.BasePath = "/"
	s.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		//ginSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", conf.GetInt("app.http.port"))),
		ginSwagger.DefaultModelsExpandDepth(-1),
		ginSwagger.PersistAuthorization(true),
	))

	s.Use(
		middleware.CORSMiddleware(),
		middleware.ResponseLogMiddleware(logger),
		middleware.RequestLogMiddleware(logger),
		//middleware.SignMiddleware(log),
	)

	// 使用 router 包注册路由
	router.InitRouter(s.Engine, jwt, e, logger)

	// 自动同步路由到数据库
	go syncRoutesToDB(s.Engine, logger)

	return s
}

// syncRoutesToDB 同步 Gin 路由到数据库
func syncRoutesToDB(engine *gin.Engine, logger *log.Logger) {
	// 获取所有注册的路由
	ginRoutes := engine.Routes()
	
	// 转换为 api.RouteInfo
	routes := make([]api.RouteInfo, 0, len(ginRoutes))
	for _, r := range ginRoutes {
		routes = append(routes, api.RouteInfo{
			Method:  r.Method,
			Path:    r.Path,
			Handler: r.Handler,
		})
	}
	
	// 调用 service 同步
	ctx := context.Background()
	if err := service.AdminServiceApp.SyncRoutesToDB(ctx, routes); err != nil {
		logger.Error("同步路由到数据库失败")
	}
}
