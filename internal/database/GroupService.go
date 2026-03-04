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
	GetGroupDetails(ctx fiber.Ctx, searchInfo string) (*model.Group, error) 
}

func(s *service)GetGroups(ctx fiber.Ctx) (*[]model.Group, error){
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil,err
	}
	if err := CheckLogin(ctx, accountID,true); err != nil {
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
	if err := CheckLogin(ctx, accountID,true); err != nil {
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

// GetGroupDetails retrieves the details of a group based on the provided search information.
// It first extracts the account ID from the JWT token in the context and validates the login status.
// Then, it performs a database transaction to fetch the group details using either the UUID or name.
// If the group is not found, it returns a "not found" error.
//
// Parameters:
//   - ctx: The Fiber context containing the request information and JWT claims.
//   - searchInfo: A string used to search for the group by either UUID or name.
//
// Returns:
//   - *model.Group: A pointer to the Group model containing the retrieved group details.
//   - error: An error if any step fails, including authentication, database query, or transaction issues.
func (s *service) GetGroupDetails(ctx fiber.Ctx, searchInfo string) (*model.Group, error) {
	// Extract the account ID from the JWT token in the context.
	accountID, err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil, err
	}

	// Validate the login status of the user.
	if err := CheckLogin(ctx, accountID,true); err != nil {
		return nil, err
	}

	// Initialize a new Group model to store the retrieved details.
	group := new(model.Group)

	// Perform a database transaction to fetch the group details.
	err = s.gdb.Transaction(func(tx *gorm.DB) error {
		// Query the group by UUID or name.
		if groupDetails, err := gorm.G[model.Group](tx).Where("uuid = ? OR name = ?", searchInfo, searchInfo).First(ctx); err != nil {
			// Handle the case where the group is not found.
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return exceptions.ErrNotFound
			}
			// Log and return other database errors.
			zaplog.Zap.Error(fmt.Sprintf("select group failed: %v", err))
			return err
		} else {
			// Populate the group model with the retrieved details.
			group.UUID = groupDetails.UUID
			group.Name = groupDetails.Name
			group.Notice = groupDetails.Notice
			group.OwnerID = groupDetails.OwnerID
			group.CreatedAt = groupDetails.CreatedAt
		}
		return nil
	})

	// Return the populated group model and any error encountered.
	return group, err
}