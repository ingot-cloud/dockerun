package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/models"
	"encoding/hex"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

type Claims struct {
	Origin      string   `json:"origin,omitempty"`
	UserID      uint     `json:"uid"`
	AuthVersion uint     `json:"ver"`
	Purpose     string   `json:"purpose"`
	Scopes      []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

func Random() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func Hash(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func Authenticate(username, password string) (*models.User, error) {
	var u models.User
	if err := database.DB.Where("username = ?", username).First(&u).Error; err != nil || !u.Enabled || !u.CheckPassword(password) {
		return nil, errors.New("用户名或密码错误")
	}
	return &u, nil
}
func Issue(u *models.User, purpose string, scopes []string, ttl time.Duration) (string, error) {
	cfg := config.Load()
	key := cfg.JWTSecret
	aud := "dockerun-web"
	origin := ""
	if purpose == "registry-import" {
		purpose = "registry"
		origin = "bundle"
	}
	if purpose == "registry" {
		key = cfg.RegistrySecret
		aud = "dockerun-registry"
	}
	if len(key) < 32 {
		return "", errors.New("signing key unavailable")
	}
	c := Claims{Origin: origin, UserID: u.ID, AuthVersion: u.AuthVersion, Purpose: purpose, Scopes: scopes, RegisteredClaims: jwt.RegisteredClaims{Issuer: "dockerun", Audience: jwt.ClaimStrings{aud}, Subject: u.Username, ID: Random(), IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(key))
}
func Validate(raw, purpose string) (*models.User, *Claims, error) {
	var c Claims
	cfg := config.Load()
	key := cfg.JWTSecret
	aud := "dockerun-web"
	if purpose == "registry" {
		key = cfg.RegistrySecret
		aud = "dockerun-registry"
	}
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) { return []byte(key), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("dockerun"), jwt.WithAudience(aud), jwt.WithExpirationRequired())
	if err != nil || c.Purpose != purpose {
		return nil, nil, errors.New("无效或过期令牌")
	}
	var u models.User
	if err = database.DB.First(&u, c.UserID).Error; err != nil || !u.Enabled || u.Password == "" || u.AuthVersion != c.AuthVersion {
		return nil, nil, errors.New("身份已失效，请重新登录")
	}
	return &u, &c, nil
}
func Automation(raw string) (*models.User, []string, error) {
	var t models.AutomationToken
	if err := database.DB.Where("hash = ? AND revoked = ?", Hash(raw), false).First(&t).Error; err != nil || time.Now().After(t.ExpiresAt) {
		return nil, nil, errors.New("自动化令牌已失效")
	}
	var u models.User
	if err := database.DB.First(&u, t.UserID).Error; err != nil || !u.Enabled || u.AuthVersion != t.AuthVersion {
		return nil, nil, errors.New("身份已失效")
	}
	return &u, strings.Fields(t.Scopes), nil
}
func Has(scopes []string, s string) bool {
	for _, v := range scopes {
		if v == s {
			return true
		}
	}
	return false
}
func Audit(uid uint, action, resource string) error {
	return database.DB.Create(&models.AuditEvent{UserID: uid, Action: action, Resource: resource}).Error
}
