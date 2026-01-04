package repository

import (
	"context"
	"go-noah/internal/model"
)

// UserRepository 用户数据访问层（简化版：直接使用结构体，不定义接口）
type UserRepository struct {
	*Repository
}

func NewUserRepository(repository *Repository) *UserRepository {
	return &UserRepository{
		Repository: repository,
	}
}

func (r *UserRepository) GetUser(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	return &user, nil
}
