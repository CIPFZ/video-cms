package jwt

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/CIPFZ/video-cms/internal/core/claims"
	"github.com/CIPFZ/video-cms/internal/core/config"
	"github.com/CIPFZ/video-cms/internal/modules/system/dto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

var (
	ErrTokenExpired     = errors.New("token已过期")
	ErrTokenNotValidYet = errors.New("token尚未激活")
	ErrTokenMalformed   = errors.New("这不是一个token")
	ErrTokenInvalid     = errors.New("无法处理此token")
)

type JWT struct {
	cfg         config.JWT
	logger      *zap.Logger
	redis       redis.UniversalClient
	signingKey  []byte
	issuer      string
	expiresTime time.Duration
	bufferTime  time.Duration
	sf          *singleflight.Group
}

func NewJWT(cfg config.JWT, logger *zap.Logger, redis redis.UniversalClient) *JWT {
	ep, _ := parseDuration(cfg.ExpiresTime)
	bf, _ := parseDuration(cfg.BufferTime)

	return &JWT{
		cfg:         cfg,
		logger:      logger,
		redis:       redis,
		signingKey:  []byte(cfg.SigningKey),
		issuer:      cfg.Issuer,
		expiresTime: ep,
		bufferTime:  bf,
		sf:          &singleflight.Group{},
	}
}

func (j *JWT) CreateClaims(baseClaims dto.BaseClaims) claims.CustomClaims {
	bfSeconds := int64(j.bufferTime / time.Second)

	return claims.CustomClaims{
		BaseClaims: baseClaims,
		BufferTime: bfSeconds,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Audience:  jwt.ClaimStrings{j.cfg.Issuer},
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.expiresTime)),
			Issuer:    j.issuer,
		},
	}
}

func (j *JWT) CreateToken(claims claims.CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.signingKey)
}

func (j *JWT) ParseToken(tokenString string) (*claims.CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.signingKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(j.issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*claims.CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func (j *JWT) ResolveToken(ctx context.Context, oldToken string, claims *claims.CustomClaims) (string, *claims.CustomClaims, error) {
	if claims.ExpiresAt.Unix()-time.Now().Unix() > claims.BufferTime {
		return oldToken, claims, nil
	}

	newToken, err, _ := j.sf.Do("JWT:"+oldToken, func() (interface{}, error) {
		return j.createTokenByOldToken(ctx, oldToken, *claims)
	})
	if err != nil {
		return "", nil, err
	}

	newClaims, err := j.ParseToken(newToken.(string))
	return newToken.(string), newClaims, err
}

func (j *JWT) createTokenByOldToken(ctx context.Context, oldToken string, c claims.CustomClaims) (string, error) {
	if j.redis != nil {
		isBlack, _ := j.redis.Get(ctx, "jwt_black:"+oldToken).Result()
		if isBlack != "" {
			return "", errors.New("token已被注销")
		}
	}

	newClaims := j.CreateClaims(c.BaseClaims)
	newClaims.ID = c.ID
	newToken, err := j.CreateToken(newClaims)
	if err != nil {
		return "", err
	}

	return newToken, nil
}

func (j *JWT) SetBlacklist(ctx context.Context, token string, expiration time.Duration) error {
	if j.redis == nil {
		return nil
	}
	return j.redis.Set(ctx, "jwt_black:"+token, "1", expiration).Err()
}

func (j *JWT) IsBlacklist(ctx context.Context, token string) bool {
	if j.redis == nil {
		return false
	}
	val, _ := j.redis.Get(ctx, "jwt_black:"+token).Result()
	return val != ""
}

func parseDuration(d string) (time.Duration, error) {
	d = strings.TrimSpace(d)
	dr, err := time.ParseDuration(d)
	if err == nil {
		return dr, nil
	}
	if strings.HasSuffix(d, "d") {
		days := strings.TrimSuffix(d, "d")
		v, err := strconv.Atoi(days)
		if err == nil {
			return time.Hour * 24 * time.Duration(v), nil
		}
	}
	return 0, errors.New("invalid duration")
}
