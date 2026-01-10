package global

import (
	"go-noah/pkg/jwt"
	"go-noah/pkg/log"
	"go-noah/pkg/sid"

	"github.com/casbin/casbin/v2"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// 全局基础设施组件（只包含基础设施，不包含业务层）
var (
	DB       *gorm.DB
	Logger   *log.Logger
	JWT      *jwt.JWT
	Sid      *sid.Sid
	Enforcer *casbin.SyncedEnforcer
	Redis    *redis.Client
	Conf     *viper.Viper
)
