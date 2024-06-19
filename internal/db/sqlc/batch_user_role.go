package db

import (
	"context"
	"encoding/json"
)

type UserRolesParams struct {
	Username string `json:"username"`
	RoleName string `json:"role_name"`
}

func (s *SQLStore) BulkCreateUserRoles(ctx context.Context, arg []UserRolesParams) (ret []UserRolesParams, errs []error) {
	data, _ := json.Marshal(arg)
	var createUserRolesParams []BatchCreateUserRolesParams
	json.Unmarshal(data, &createUserRolesParams)
	s.BatchCreateUserRoles(ctx, createUserRolesParams).QueryRow(func(i int, ru RoleUser, err error) {
		if err != nil {
			errs = append(errs, err)
		} else {
			ret = append(ret, UserRolesParams{
				Username: arg[i].Username,
				RoleName: arg[i].RoleName,
			})
		}
	})
	return
}

func (s *SQLStore) BulkDeleteUserRoles(ctx context.Context, arg []UserRolesParams) []error {
	data, _ := json.Marshal(arg)
	var deleteUserRolesParams []BatchDeleteUserRolesParams
	json.Unmarshal(data, &deleteUserRolesParams)
	var errs []error
	s.BatchDeleteUserRoles(ctx, deleteUserRolesParams).Exec(func(i int, err error) {
		if err != nil {
			errs = append(errs, err)
		}
	})
	return errs
}
