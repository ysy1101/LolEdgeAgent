package handler

import (
	"net/http"
	"os"
	"time"

	"loledgeagent/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct{ db *gorm.DB }

func NewAuthHandler(db *gorm.DB) *AuthHandler { return &AuthHandler{db: db} }

func jwtSecret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "loledgeagent-dev-secret"
	}
	return []byte(s)
}

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if len(req.Username) < 2 || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名至少2位，密码至少6位"})
		return
	}

	// 检查用户名是否已存在
	var exist models.User
	if h.db.Where("username = ?", req.Username).First(&exist).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "用户名已存在"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "密码加密失败"})
		return
	}

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := h.db.Create(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 自动创建偏好
	h.db.Create(&models.Preference{UserID: user.ID})

	token, _ := generateToken(user)
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username},
	}})
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var user models.User
	if h.db.Where("username = ?", req.Username).First(&user).Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	token, _ := generateToken(&user)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username},
	}})
}

func generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}
