package application

import (
	"context"

	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/entity"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/repository"
)

type userApp struct {
	user repository.UserRepository
}

// userApp implements the UserInterface
var _ UserInterface = &userApp{}

type UserInterface interface {
	ListUser(ctx context.Context) ([]entity.User, error)
	GetUserByID(ctx context.Context, id int) (entity.User, error)
	GetUserByOpenID(ctx context.Context, openID string) (entity.User, error)
	UpdateUser(ctx context.Context, user entity.User) error
	SendSms(ctx context.Context, req entity.SendSmsReq) error
	VerifySmsCode(ctx context.Context, req entity.VerifyCodeReq) (bool, bool, error)
}

func (u *userApp) ListUser(ctx context.Context) ([]entity.User, error) {
	return u.user.ListUser(ctx)
}

func (u *userApp) GetUserByID(ctx context.Context, id int) (entity.User, error) {
	return u.user.GetUserByID(ctx, id)
}

func (u *userApp) GetUserByOpenID(ctx context.Context, openID string) (entity.User, error) {
	return u.user.GetUserByOpenID(ctx, openID)
}

func (u *userApp) UpdateUser(ctx context.Context, user entity.User) error {
	return u.user.UpdateUser(ctx, user)
}

func (u *userApp) SendSms(ctx context.Context, req entity.SendSmsReq) error {
	return u.user.SendSms(ctx, req)
}

func (u *userApp) VerifySmsCode(ctx context.Context, req entity.VerifyCodeReq) (bool, bool, error) {
	return u.user.VerifySmsCode(ctx, req)
}
