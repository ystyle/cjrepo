package auth

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserType string `json:"user_type"` // admin
	jwt.RegisteredClaims
}

// AuthService 认证服务
type AuthService struct {
	adminKeyMD5 string // 管理密钥的 MD5 值
	jwtSecret   []byte
}

// NewAuthService 创建认证服务
func NewAuthService(adminKey string) *AuthService {
	// 对管理密钥进行 MD5 加密
	hash := md5.Sum([]byte(adminKey))
	return &AuthService{
		adminKeyMD5: hex.EncodeToString(hash[:]),
		jwtSecret:   []byte("cjrepo-jwt-secret-key-2024"), // JWT 签名密钥
	}
}

// VerifyAdminKey 验证管理密钥（比较 MD5）
func (s *AuthService) VerifyAdminKey(keyMD5 string) bool {
	return s.adminKeyMD5 == keyMD5
}

// GenerateToken 生成 JWT token
func (s *AuthService) GenerateToken() (string, error) {
	claims := Claims{
		UserType: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)), // 30分钟过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "cjrepo",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken 验证 JWT token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// HashMD5 对字符串进行 MD5 加密
func HashMD5(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:])
}
