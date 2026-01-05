package noah

import (
	"go-noah/internal/job"
	"go-noah/internal/repository"
	"go-noah/internal/server"
	"go-noah/internal/task"
	"go-noah/pkg/app"
	"go-noah/pkg/global"
	"go-noah/pkg/jwt"
	"go-noah/pkg/log"
	"go-noah/pkg/sid"

	"github.com/spf13/viper"
)

func NewServerApp(conf *viper.Viper, logger *log.Logger) (*app.App, func(), error) {
	// 只初始化基础设施（存入 global）
	global.Sid = sid.NewSid()
	global.JWT = jwt.NewJwt(conf)
	global.DB = repository.NewDB(conf, logger)
	global.Enforcer = repository.NewCasbinEnforcer(conf, logger, global.DB)
	global.Logger = logger
	global.Conf = conf

	// 创建 Repository 和 Transaction（不存储在 global，避免循环导入）
	repo := repository.NewRepository(logger, global.DB, global.Enforcer)
	transaction := repository.NewTransaction(repo)

	httpServer := server.NewHTTPServer(logger, conf, global.JWT, global.Enforcer)

	jobBase := job.NewJob(transaction, global.Logger, global.Sid)
	userRepo := repository.NewUserRepository(repo)
	userJob := job.NewUserJob(jobBase, userRepo)
	jobServer := server.NewJobServer(global.Logger, userJob)

	a := app.NewApp(
		app.WithServer(httpServer, jobServer),
		app.WithName("noah-server"),
	)

	cleanup := func() {
		sqlDB, _ := global.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	return a, cleanup, nil
}

func NewTaskApp(conf *viper.Viper, logger *log.Logger) (*app.App, func(), error) {
	// 只初始化基础设施（存入 global）
	global.Sid = sid.NewSid()
	global.DB = repository.NewDB(conf, logger)
	global.Enforcer = repository.NewCasbinEnforcer(conf, logger, global.DB)
	global.Logger = logger
	global.Conf = conf

	// 创建 Repository 和 Transaction（不存储在 global，避免循环导入）
	repo := repository.NewRepository(logger, global.DB, global.Enforcer)
	transaction := repository.NewTransaction(repo)

	userRepo := repository.NewUserRepository(repo)

	tk := task.NewTask(transaction, global.Logger, global.Sid)
	userTask := task.NewUserTask(tk, userRepo)
	taskServer := server.NewTaskServer(global.Logger, userTask)

	a := app.NewApp(
		app.WithServer(taskServer),
		app.WithName("noah-task"),
	)
	cleanup := func() {
		sqlDB, _ := global.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	return a, cleanup, nil
}

func NewMigrateApp(conf *viper.Viper, logger *log.Logger) (*app.App, func(), error) {
	// 只初始化基础设施（存入 global）
	global.Sid = sid.NewSid()
	global.DB = repository.NewDB(conf, logger)
	global.Enforcer = repository.NewCasbinEnforcer(conf, logger, global.DB)
	global.Logger = logger
	global.Conf = conf

	migrateServer := server.NewMigrateServer(global.DB, global.Logger, global.Sid, global.Enforcer)

	a := app.NewApp(
		app.WithServer(migrateServer),
		app.WithName("noah-migrate"),
	)
	cleanup := func() {
		sqlDB, _ := global.DB.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
	return a, cleanup, nil
}
