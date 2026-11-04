package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/datatypes"

	"github.com/CIPFZ/video-cms/internal/core/claims"
	logger "github.com/CIPFZ/video-cms/internal/core/log"
	"github.com/CIPFZ/video-cms/internal/core/session"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/CIPFZ/video-cms/internal/modules/system/model"
	"github.com/CIPFZ/video-cms/internal/modules/system/repository"
	"github.com/CIPFZ/video-cms/internal/svc"
	"github.com/CIPFZ/video-cms/pkg/utils"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IUserService interface {
	Register(ctx context.Context, req dto.RegisterReq) (*model.SysUser, error)
	Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResponse, error)
	GetUserInfo(ctx context.Context, userUUID uuid.UUID) (*model.SysUser, error)
	generateJwtToken(ctx context.Context, user *model.SysUser) (string, claims.CustomClaims, error)
	Logout(ctx context.Context, token string) error
	GetUserList(ctx context.Context, req dto.SearchUserReq) (list []model.SysUser, total int64, err error)
	AddUser(ctx context.Context, req dto.AddUserReq) error
	UpdateUser(ctx context.Context, req dto.UpdateUserReq) error
	SwitchAuthority(ctx context.Context, uuid uuid.UUID, authorityId uint) (*dto.LoginResponse, error)
	DeleteUser(ctx context.Context, id uint) error
	ResetPassword(ctx context.Context, req dto.ResetPasswordReq) error
	UpdateSelfInfo(ctx context.Context, uid uuid.UUID, req dto.UpdateSelfInfoReq) error
	UpdateUiConfig(ctx context.Context, uid uuid.UUID, req dto.UpdateUiConfigReq) error
	UpdateAvatar(ctx context.Context, uid uuid.UUID, avatarUrl string) error
}

type UserService struct {
	svcCtx   *svc.ServiceContext
	userRepo repository.IUserRepository
}

// NewUserService 构造函数
// 注意：这里我们传入 repo
func NewUserService(svcCtx *svc.ServiceContext, userRepo repository.IUserRepository) IUserService {
	return &UserService{
		svcCtx:   svcCtx,
		userRepo: userRepo,
	}
}

// Register 用户注册实现
func (s *UserService) Register(ctx context.Context, req dto.RegisterReq) (*model.SysUser, error) {
	if !s.svcCtx.Config.System.AllowRegistration {
		return nil, errors.New("公开注册未开启，请联系管理员")
	}
	log := logger.GetLogger(ctx)
	_, searchErr := s.userRepo.FindByUsername(ctx, req.Username)

	if searchErr == nil {
		return nil, errors.New("用户名已存在")
	}
	if !errors.Is(searchErr, gorm.ErrRecordNotFound) {
		s.svcCtx.Logger.Error("查询用户失败", zap.Error(searchErr))
		return nil, errors.New("系统内部错误")
	}

	// 2. 密码加密
	hashPwd, err := utils.BcryptHash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %v", err)
	}

	// 3. 构建用户模型
	newUser := model.SysUser{
		Username:    req.Username,
		Password:    hashPwd,
		NickName:    req.NickName,
		Phone:       req.Phone,
		Email:       req.Email,
		Avatar:      model.DefaultUserAvatar,
		Status:      model.UserActive, // 默认正常
		AuthorityID: model.DefaultUserAuthorityID,
		Authorities: []model.SysAuthority{{AuthorityId: model.DefaultUserAuthorityID}},
		UUID:        uuid.New(),
	}

	// 4. 插入数据库
	if insertErr := s.userRepo.Create(ctx, &newUser); insertErr != nil {
		log.Error("create user failed", zap.Error(insertErr))
		return nil, errors.New("注册失败，请稍后重试")
	}

	return &newUser, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req dto.LoginReq) (*dto.LoginResponse, error) {
	log := logger.GetLogger(ctx)
	// 1. 查询用户 (这里不需要 Preload 太多关联表了，因为不返回用户信息，只需验证密码)
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	// 验证用户是否存在、密码比对、状态检查逻辑保持不变
	if err != nil {
		log.Error("login_failed_user_not_found", zap.Error(err), zap.String("username", req.Username))
		return nil, errors.New("用户名或密码错误")
	}

	if !utils.BcryptCheck(req.Password, user.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	if user.Status != model.UserActive {
		return nil, errors.New("此用户已经被禁用")
	}

	// 2. 签发 Token
	token, c, err := s.generateJwtToken(ctx, user)
	if err != nil {
		log.Error("generate_token_failed", zap.Error(err))
		return nil, errors.New("获取Token失败")
	}

	// 3. 只返回 Token 和 过期时间
	return &dto.LoginResponse{
		Token:     token,
		ExpiresAt: c.RegisteredClaims.ExpiresAt.Unix() * 1000,
	}, nil
}

// generateJwtToken 内部辅助函数
func (s *UserService) generateJwtToken(ctx context.Context, user *model.SysUser) (string, claims.CustomClaims, error) {
	// 构造 Claims
	customClaims := s.svcCtx.JWT.CreateClaims(dto.BaseClaims{
		UUID:         user.UUID,
		UserID:       user.ID,
		NickName:     user.NickName,
		Username:     user.Username,
		AuthorityId:  user.AuthorityID,
		TokenVersion: user.TokenVersion,
	})

	token, err := s.svcCtx.JWT.CreateToken(customClaims)
	if err != nil {
		return "", customClaims, err
	}
	err = s.svcCtx.Sessions.Create(ctx, session.Record{ID: customClaims.ID, UserID: user.ID, TokenVersion: user.TokenVersion, ExpiresAt: customClaims.ExpiresAt.Time})
	return token, customClaims, err
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, userUUID uuid.UUID) (*model.SysUser, error) {
	log := logger.GetLogger(ctx)
	user, err := s.userRepo.FindByUuid(ctx, userUUID)

	if err != nil {
		log.Error("get_user_info_failed", zap.Error(err), zap.Any("userUUID", userUUID))
		return nil, err
	}

	// 安全起见，清空密码
	user.Password = ""
	return user, nil
}

// Logout 用户登出实现
func (s *UserService) Logout(ctx context.Context, token string) error {
	c, err := s.svcCtx.JWT.ParseToken(token)
	if err != nil {
		return nil
	}
	return s.svcCtx.Sessions.Revoke(ctx, c.ID)
}

// GetUserList 分页获取用户列表
func (s *UserService) GetUserList(ctx context.Context, req dto.SearchUserReq) (list []model.SysUser, total int64, err error) {
	return s.userRepo.GetList(ctx, req)
}

// AddUser 新增用户
func (s *UserService) AddUser(ctx context.Context, req dto.AddUserReq) error {
	// 1. 查重
	_, searchErr := s.userRepo.FindByUsername(ctx, req.Username)
	if !errors.Is(searchErr, gorm.ErrRecordNotFound) {
		return errors.New("用户名已存在")
	}

	// 1. 检查角色列表
	if len(req.AuthorityIds) == 0 {
		return errors.New("请至少选择一个角色")
	}

	// 2. 加密密码
	hashPwd, err := utils.BcryptHash(req.Password)
	if err != nil {
		return err
	}

	// 3. 构建用户
	newUser := model.SysUser{
		UUID:        uuid.New(),
		Username:    req.Username,
		Password:    hashPwd,
		NickName:    req.NickName,
		Avatar:      model.DefaultUserAvatar,
		AuthorityID: req.AuthorityIds[0],
		Phone:       req.Phone,
		Email:       req.Email,
		Status:      model.UserActive,
	}

	if req.Status != nil {
		newUser.Status = *req.Status
	}
	// 4. 处理多角色
	var auths []model.SysAuthority
	for _, id := range req.AuthorityIds {
		auths = append(auths, model.SysAuthority{AuthorityId: id})
	}
	newUser.Authorities = auths

	return s.userRepo.Create(ctx, &newUser)
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(ctx context.Context, req dto.UpdateUserReq) error {
	user, searchErr := s.userRepo.FindById(ctx, req.ID)
	if searchErr != nil {
		return errors.New("用户不存在")
	}

	if len(req.AuthorityIds) <= 0 {
		return errors.New("请至少选择一个角色")
	}

	return s.userRepo.UpdateWithRoles(ctx, user, req)
}

// SwitchAuthority 切换角色
func (s *UserService) SwitchAuthority(ctx context.Context, uuid uuid.UUID, authorityId uint) (*dto.LoginResponse, error) {
	user, searchErr := s.userRepo.FindByUuid(ctx, uuid)
	if searchErr != nil {
		return nil, searchErr
	}

	// 2. 校验：用户是否真的拥有目标角色？
	hasAuth := false
	for _, auth := range user.Authorities {
		if auth.AuthorityId == authorityId {
			hasAuth = true
			break
		}
	}
	if !hasAuth {
		return nil, errors.New("您未拥有该角色权限")
	}

	// 3. 更新数据库中的 "当前角色"
	if err := s.userRepo.Update(ctx, user, map[string]interface{}{"authority_id": authorityId, "token_version": gorm.Expr("token_version + 1")}); err != nil {
		return nil, err
	}

	// 4. 更新内存对象以便签发 Token
	user, searchErr = s.userRepo.FindByUuid(ctx, uuid)
	if searchErr != nil {
		return nil, searchErr
	}

	// 5. 签发新 Token (因为 Token 里包含 AuthorityId，切换角色必须换 Token)
	token, c, err := s.generateJwtToken(ctx, user)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token:     token,
		ExpiresAt: c.RegisteredClaims.ExpiresAt.Unix() * 1000,
	}, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.userRepo.DeleteWithAssociations(ctx, id)
}

// ResetPassword 重置密码
func (s *UserService) ResetPassword(ctx context.Context, req dto.ResetPasswordReq) error {
	hashPwd, err := utils.BcryptHash(req.Password)
	if err != nil {
		return err
	}
	return s.userRepo.ResetPassword(ctx, req.ID, hashPwd)
}

// UpdateSelfInfo 更新个人基础信息
func (s *UserService) UpdateSelfInfo(ctx context.Context, uid uuid.UUID, req dto.UpdateSelfInfoReq) error {
	updates := map[string]interface{}{
		"nick_name": req.NickName,
		"bio":       req.Bio,
	}
	return s.userRepo.UpdateColumn(ctx, uid, updates)
}

// UpdateUiConfig 更新界面配置 (存入 settings 字段)
func (s *UserService) UpdateUiConfig(ctx context.Context, uid uuid.UUID, req dto.UpdateUiConfigReq) error {
	// 将 map 转为 JSON 字节，再转为 datatypes.JSON
	// GORM 的 datatypes.JSON 需要 []byte
	jsonBytes, err := json.Marshal(req.Settings)
	if err != nil {
		return err
	}

	return s.userRepo.UpdateColumn(ctx, uid, map[string]interface{}{
		"settings": datatypes.JSON(jsonBytes),
	})
}

// UpdateAvatar 更新用户头像
func (s *UserService) UpdateAvatar(ctx context.Context, uid uuid.UUID, avatarUrl string) error {
	// 直接更新指定字段，高效且安全
	return s.userRepo.UpdateColumn(ctx, uid, map[string]interface{}{
		"avatar": avatarUrl,
	})
}
