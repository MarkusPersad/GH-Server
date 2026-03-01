package database

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FriendService interface {
	AddFriend(request *request.AddFriendRequest,ctx fiber.Ctx) error
	AgreeFriend(request *request.AddFriendRequest,ctx fiber.Ctx) error
}

func(s *service)AddFriend(request *request.AddFriendRequest,ctx fiber.Ctx) error {
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return err
	}
	if err := CheckLogin(ctx, accountID); err != nil {
		return  err
	}
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		userId,err := uuid.Parse(request.UserId)
		if err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Parse userId failed: %v", err))
			return err
		}

		if _,err := gorm.G[model.User](tx).Where("uuid = ?",userId).First(ctx);err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				return exceptions.ErrNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		}

		account,faccount:= utils.StringSwitch(accountID,request.UserId)
	
		if _,err := gorm.G[model.UserFriend](tx).Where("user_id = ? AND friend_id = ?",account,faccount).First(ctx);err != nil {
			if !errors.Is(err,gorm.ErrRecordNotFound) {
				zaplog.Zap.Error(fmt.Sprintf("select user_friend failed: %v", err))
				return err
			}
		} else {
			return exceptions.ErrFriendAlreadyExists
		}
		return  gorm.G[model.UserFriend](tx).Create(ctx,&model.UserFriend{
			UserID: account,
			FriendID: faccount,
		})
	})
}

func(s *service)AgreeFriend(request *request.AddFriendRequest,ctx fiber.Ctx) error {
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return err
	}
	if err := CheckLogin(ctx, accountID); err != nil {
		return  err
	}
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		account,faccount := utils.StringSwitch(accountID,request.UserId)
		if _,err := gorm.G[model.UserFriend](tx).Where("user_id = ? AND friend_id = ?",account,faccount).First(ctx);err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				return exceptions.ErrNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user_friend failed: %v", err))
			return err
		}

		if _,err :=  gorm.G[model.UserFriend](tx).Where("user_id = ? AND friend_id = ?",account,faccount).Select("status","version").Updates(ctx,model.UserFriend{
			Status: 0,
		});err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user_friend failed: %v", err))
			return err
		}
		return nil
	}) 
}
