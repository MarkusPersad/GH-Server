package database

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type GroupService interface {
	GetGroups(ctx fiber.Ctx) (*[]model.Group, error) 
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
			 Joins("LEFT JOIN `group` ON group_member.group_id = group.group_id").
			 Where("group_member.user_id = ?", accountID).
			 Scan(groups).Error
	})
	return groups,err
}