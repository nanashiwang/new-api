package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// SetContentSafetyWhitelist serializes with violation recording on the user row.
// History and account status are deliberately preserved when toggling exemption.
func SetContentSafetyWhitelist(userID, adminID int, enabled bool) error {
	if userID <= 0 || adminID <= 0 {
		return errors.New("无效的白名单用户")
	}
	changed := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", userID).UpdateColumn("status", gorm.Expr("status")).Error; err != nil {
			return err
		}
		var admin User
		if err := tx.Select("id", "role", "status").First(&admin, adminID).Error; err != nil {
			return err
		}
		if admin.Status != common.UserStatusEnabled || admin.Role < common.RoleAdminUser {
			return errors.New("仅管理员可修改内容安全白名单")
		}
		var user User
		if err := tx.Select("id", "role", "content_safety_whitelisted").First(&user, userID).Error; err != nil {
			return err
		}
		if admin.Role != common.RoleRootUser && admin.Role <= user.Role {
			return errors.New("无权修改同级或更高级用户的白名单")
		}
		if user.ContentSafetyWhitelisted == enabled {
			return nil
		}
		changed = true
		return tx.Model(&User{}).Where("id = ?", userID).Update("content_safety_whitelisted", enabled).Error
	})
	if err == nil && changed {
		action := "移出"
		if enabled {
			action = "加入"
		}
		RecordLogWithAdminInfo(userID, LogTypeManage, fmt.Sprintf("管理员将用户%s内容安全白名单；历史记录保留，账号启停状态不变", action), map[string]interface{}{"admin_id": adminID})
	}
	return err
}
