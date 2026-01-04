package service

import (
	"context"
	"go-noah/internal/model"
	"go-noah/internal/repository"
	"go-noah/pkg/global"
)

// UserService 用户业务逻辑层
type UserService struct{}

var UserServiceApp = new(UserService)

// getUserRepo 获取 UserRepository（在方法内部创建）
func (s *UserService) getUserRepo() *repository.UserRepository {
	return repository.NewUserRepository(global.Repo)
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	repo := s.getUserRepo()
	return repo.GetUser(ctx, id)
}
