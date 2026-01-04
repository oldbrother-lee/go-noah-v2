package handler

import (
	"github.com/gin-gonic/gin"
)

// UserHandler 用户 Handler
type UserHandler struct{}

// UserHandlerApp 全局 Handler 实例
var UserHandlerApp = new(UserHandler)

func (h *UserHandler) GetUsers(ctx *gin.Context) {
	// 直接使用全局 Service
	// user, err := service.UserServiceApp.GetUser(ctx, id)
}
