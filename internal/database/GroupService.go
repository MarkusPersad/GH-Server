package database

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GroupService interface {
	GetGroups(ctx fiber.Ctx) (*[]model.Group, error)
	CreateGroup(request *request.GroupCreateRequest,ctx fiber.Ctx) error 
}

func(s *service)GetGroups(ctx fiber.Ctx) (*[]model.Group, error){
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil,err
	}
	if err := CheckLogin(ctx, accountID); err != nil {
		return  nil,err
	}
	groups := new([]model.Group)
	err = s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("uuid = ?",accountID).First(ctx); err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				return exceptions.ErrNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		}
		return tx.Table("group_member").
			 Select("group.group_id,group.created_at, group.name, group.notice").
			 Joins("LEFT JOIN `group` ON group_member.group_id = group.uuid").
			 Where("group_member.user_id = ?", accountID).
			 Scan(groups).Error
	})
	return groups,err
}

func(s *service)CreateGroup(request *request.GroupCreateRequest,ctx fiber.Ctx) error{
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return err
	}
	if err := CheckLogin(ctx, accountID); err != nil {
		return  err
	}
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("uuid = ?",accountID).First(ctx);err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				return exceptions.ErrNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		}
		groupUUID := uuid.NewSHA1(uuid.NameSpaceX500,[]byte(request.Name))
		if err := gorm.G[model.Group](tx).Create(ctx,&model.Group{
			UUID: groupUUID.String(),
			OwnerID: accountID,
			Name: request.Name,
			Notice: request.Notice,
		});err != nil {
			if errors.Is(err,gorm.ErrDuplicatedKey) {
				return exceptions.ErrGroupAlreadyExists
			}
			zaplog.Zap.Error(fmt.Sprintf("insert group failed: %v", err))
			return err
		}
		if err := gorm.G[model.GroupMember](tx).Create(ctx,&model.GroupMember{
			UserId: accountID,
			GroupId: groupUUID.String(),
		}); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("insert group member failed: %v", err))
			return err
		}
		return nil
	}) 
}