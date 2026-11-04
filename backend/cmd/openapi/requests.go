package main

import (
	"github.com/CIPFZ/video-cms/internal/modules/common"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"reflect"
	"strings"
)

var requests = map[string]any{
	"/user/login": dto.LoginReq{}, "/user/register": dto.RegisterReq{},
	"/sys/user/getUserList": dto.SearchUserReq{}, "/sys/user/addUser": dto.AddUserReq{}, "/sys/user/updateUser": dto.UpdateUserReq{}, "/sys/user/resetPassword": dto.ResetPasswordReq{}, "/sys/user/info": dto.UpdateSelfInfoReq{}, "/sys/user/ui-config": dto.UpdateUiConfigReq{}, "/sys/user/avatar": dto.UpdateAvatarReq{},
	"/sys/menu/addBaseMenu": dto.AddMenuReq{}, "/sys/menu/updateBaseMenu": dto.UpdateMenuReq{}, "/sys/menu/getMenuAuthority": dto.GetAuthorityIdReq{},
	"/sys/authority/getAuthorityList": common.PageInfo{}, "/sys/authority/createAuthority": dto.CreateAuthorityReq{}, "/sys/authority/updateAuthority": dto.UpdateAuthorityReq{}, "/sys/authority/setAuthorityMenus": dto.SetAuthorityMenusReq{},
	"/sys/api/getApiList": dto.SearchApiReq{}, "/sys/api/createApi": dto.CreateApiReq{}, "/sys/api/updateApi": dto.UpdateApiReq{}, "/sys/api/deleteApi": dto.DeleteApiReq{},
	"/sys/api-token/getApiTokenList": dto.SearchApiTokenReq{}, "/sys/api-token/create": dto.CreateApiTokenReq{}, "/sys/api-token/update": dto.UpdateApiTokenReq{}, "/sys/api-token/delete": dto.DeleteApiTokenReq{}, "/sys/api-token/reset": dto.ToggleApiTokenReq{}, "/sys/api-token/enable": dto.ToggleApiTokenReq{}, "/sys/api-token/disable": dto.ToggleApiTokenReq{},
	"/sys/casbin/getPolicyPathByAuthorityId": dto.GetPolicyPathByAuthorityIdReq{}, "/sys/casbin/updateCasbin": dto.UpdateCasbinReq{},
	"/sys/notice/createNotice": dto.CreateNoticeReq{}, "/sys/notice/getNoticeList": dto.SearchNoticeReq{}, "/sys/notice/markRead": dto.MarkNoticeReadReq{},
	"/sys/operationLog/getOperationLogList": dto.SearchOperationLogReq{}, "/sys/operationLog/deleteOperationLogByIds": dto.DeleteOperationLogReq{},
}

func requestSchema(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": requestSchema(t.Elem())}
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Anonymous {
				embedded := requestSchema(f.Type)
				for k, v := range embedded["properties"].(map[string]any) {
					props[k] = v
				}
				if r, ok := embedded["required"].([]string); ok {
					required = append(required, r...)
				}
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			props[name] = requestSchema(f.Type)
			for _, rule := range strings.Split(f.Tag.Get("binding"), ",") {
				if rule == "required" {
					required = append(required, name)
				}
			}
		}
		result := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			result["required"] = required
		}
		return result
	default:
		return map[string]any{"type": "object"}
	}
}
