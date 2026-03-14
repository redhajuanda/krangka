package libctx

import (
	"context"

	"gitlab.sicepat.tech/pka/sds/configs"
)

func RoleIsSuperadmin(ctx context.Context, cfg *configs.Config) bool {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	roleID := claims.RoleID.String()
	return roleID == cfg.Roles.Superadmin
}

func GetRoleID(ctx context.Context) string {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims.RoleID.String()
}

func GetRoleName(ctx context.Context) string {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims.RoleName
}

func GetBranchID(ctx context.Context) string {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims.BranchID.String()
}

func GetBranchName(ctx context.Context) string {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return claims.BranchName
}

func GetBranch(ctx context.Context) Branch {
	claims, err := GetClaims(ctx)
	if err != nil {
		panic(err)
	}
	return Branch{
		ID:   claims.BranchID.String(),
		Name: claims.BranchName,
	}
}

type Role struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Branch *Branch `json:"branch"`
}

func (r Role) GetRoleMap() map[string]interface{} {
	roleMap := map[string]interface{}{
		"id":   r.ID,
		"name": r.Name,
	}
	return roleMap
}

func (r Role) GetBranchMap() map[string]interface{} {
	if r.Branch == nil {
		return map[string]interface{}{}
	}
	branchMap := map[string]interface{}{
		"id":   r.Branch.ID,
		"name": r.Branch.Name,
	}
	return branchMap
}

type Branch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetRole(ctx context.Context, cfg *configs.Config) Role {
	role := Role{
		ID:   GetRoleID(ctx),
		Name: GetRoleName(ctx),
	}

	return role
}