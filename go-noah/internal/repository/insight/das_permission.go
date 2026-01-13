package insight

import (
	"context"
	"go-noah/internal/model/insight"
	"gorm.io/gorm"
)

// ============ 权限模板管理 ============

// GetPermissionTemplates 获取权限模板列表
func (r *InsightRepository) GetPermissionTemplates(ctx context.Context) ([]insight.DASPermissionTemplate, error) {
	var templates []insight.DASPermissionTemplate
	if err := r.DB(ctx).Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetPermissionTemplate 获取权限模板详情
func (r *InsightRepository) GetPermissionTemplate(ctx context.Context, id uint) (*insight.DASPermissionTemplate, error) {
	var template insight.DASPermissionTemplate
	if err := r.DB(ctx).Where("id = ?", id).First(&template).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

// CreatePermissionTemplate 创建权限模板
func (r *InsightRepository) CreatePermissionTemplate(ctx context.Context, template *insight.DASPermissionTemplate) error {
	return r.DB(ctx).Create(template).Error
}

// UpdatePermissionTemplate 更新权限模板
func (r *InsightRepository) UpdatePermissionTemplate(ctx context.Context, template *insight.DASPermissionTemplate) error {
	return r.DB(ctx).Save(template).Error
}

// DeletePermissionTemplate 删除权限模板（软删除）
func (r *InsightRepository) DeletePermissionTemplate(ctx context.Context, id uint) error {
	return r.DB(ctx).Delete(&insight.DASPermissionTemplate{}, id).Error
}

// ============ 角色权限管理 ============

// GetRolePermissions 获取角色权限列表
func (r *InsightRepository) GetRolePermissions(ctx context.Context, role string) ([]insight.DASRolePermission, error) {
	var perms []insight.DASRolePermission
	if err := r.DB(ctx).Where("role = ?", role).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// CreateRolePermission 创建角色权限
func (r *InsightRepository) CreateRolePermission(ctx context.Context, perm *insight.DASRolePermission) error {
	return r.DB(ctx).Create(perm).Error
}

// DeleteRolePermission 删除角色权限（软删除）
func (r *InsightRepository) DeleteRolePermission(ctx context.Context, id uint) error {
	return r.DB(ctx).Delete(&insight.DASRolePermission{}, id).Error
}

// BatchCreateRolePermissions 批量创建角色权限
func (r *InsightRepository) BatchCreateRolePermissions(ctx context.Context, perms []insight.DASRolePermission) error {
	if len(perms) == 0 {
		return nil
	}
	return r.DB(ctx).Create(&perms).Error
}

// ============ 权限展开和查询 ============

// ExpandRolePermissions 展开角色权限（将模板/组展开为具体权限对象）
func (r *InsightRepository) ExpandRolePermissions(ctx context.Context, role string) ([]insight.PermissionObject, error) {
	// 1. 获取角色权限
	rolePerms, err := r.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, err
	}

	var result []insight.PermissionObject

	// 2. 展开权限
	for _, perm := range rolePerms {
		switch perm.PermissionType {
		case insight.PermissionTypeObject:
			// 直接权限对象
			result = append(result, insight.PermissionObject{
				InstanceID: perm.InstanceID,
				Schema:     perm.Schema,
				Table:      perm.Table,
			})

		case insight.PermissionTypeTemplate:
			// 展开权限模板
			template, err := r.GetPermissionTemplate(ctx, perm.PermissionID)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // 模板不存在，跳过
				}
				return nil, err
			}
			result = append(result, template.Permissions...)
		}
	}

	return result, nil
}

// GetUserEffectivePermissions 获取用户实际生效的权限（展开所有角色权限）
func (r *InsightRepository) GetUserEffectivePermissions(ctx context.Context, username string) ([]insight.PermissionObject, error) {
	// 1. 从 Casbin 获取用户角色
	enforcer := r.Enforcer()
	roles, err := enforcer.GetRolesForUser(username)
	if err != nil {
		return nil, err
	}

	// 2. 收集所有角色的权限
	permissionMap := make(map[string]insight.PermissionObject) // 使用 map 去重

	for _, role := range roles {
		perms, err := r.ExpandRolePermissions(ctx, role)
		if err != nil {
			return nil, err
		}

		// 去重：使用 instance_id + schema + table 作为 key
		for _, perm := range perms {
			key := perm.InstanceID + ":" + perm.Schema + ":" + perm.Table
			permissionMap[key] = perm
		}
	}

	// 3. 转换为数组
	result := make([]insight.PermissionObject, 0, len(permissionMap))
	for _, perm := range permissionMap {
		result = append(result, perm)
	}

	return result, nil
}

