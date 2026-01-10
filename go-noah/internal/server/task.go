package server

import (
	"context"
	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go-noah/internal/task"
	"go-noah/pkg/log"
	"time"
)

type TaskServer struct {
	log         *log.Logger
	scheduler   *gocron.Scheduler
	userTask    *task.UserTask
	insightTask *task.InsightTask
	conf        *viper.Viper
}

func NewTaskServer(
	log *log.Logger,
	userTask *task.UserTask,
	insightTask *task.InsightTask,
	conf *viper.Viper,
) *TaskServer {
	return &TaskServer{
		log:         log,
		userTask:    userTask,
		insightTask: insightTask,
		conf:        conf,
	}
}
func (t *TaskServer) Start(ctx context.Context) error {
	gocron.SetPanicHandler(func(jobName string, recoverData interface{}) {
		t.log.Error("TaskServer Panic", zap.String("job", jobName), zap.Any("recover", recoverData))
	})

	// eg: crontab task
	t.scheduler = gocron.NewScheduler(time.UTC)
	// if you are in China, you will need to change the time zone as follows
	// t.scheduler = gocron.NewScheduler(time.FixedZone("PRC", 8*60*60))

	// 用户检查任务（示例，可根据需要调整或删除）
	_, err := t.scheduler.CronWithSeconds("0/3 * * * * *").Do(func() {
		err := t.userTask.CheckUser(ctx)
		if err != nil {
			t.log.Error("CheckUser error", zap.Error(err))
		}
	})
	if err != nil {
		t.log.Error("注册 CheckUser 任务失败", zap.Error(err))
	}

	// 同步数据库元数据任务
	if t.insightTask != nil {
		syncCron := t.conf.GetString("crontab.sync_db_metas")
		if syncCron == "" {
			syncCron = "*/5 * * * *" // 默认每5分钟
		}
		_, err = t.scheduler.Cron(syncCron).Do(func() {
			t.log.Info("开始执行同步数据库元数据任务", zap.Time("time", time.Now()))
			err := t.insightTask.SyncDBMeta(ctx)
			if err != nil {
				t.log.Error("同步数据库元数据失败", zap.Error(err))
			} else {
				t.log.Info("同步数据库元数据完成", zap.Time("time", time.Now()))
			}
		})
		if err != nil {
			t.log.Error("注册同步数据库元数据任务失败", zap.Error(err))
		} else {
			t.log.Info("已注册同步数据库元数据任务", zap.String("cron", syncCron))
		}
	}

	t.scheduler.StartBlocking()
	return nil
}
func (t *TaskServer) Stop(ctx context.Context) error {
	t.scheduler.Stop()
	t.log.Info("TaskServer stop...")
	return nil
}
