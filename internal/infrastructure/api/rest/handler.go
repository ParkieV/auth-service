package rest

import (
	"errors"
	emailErrors "github.com/ParkieV/auth-service/internal/domain"
	"github.com/ParkieV/auth-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	registerUC *usecase.RegisterUsecase
	loginUC    *usecase.LoginUsecase
	refreshUC  *usecase.RefreshUsecase
}

func RegisterHandlers(
	router *gin.Engine,
	registerUC *usecase.RegisterUsecase,
	loginUC *usecase.LoginUsecase,
	refreshUC *usecase.RefreshUsecase,
) {
	h := &Handler{registerUC, loginUC, refreshUC}
	api := router.Group("/api")
	{
		api.POST("/register", h.Register)
		api.POST("/login", h.Login)
		api.POST("/refresh", h.Refresh)
	}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type registerResponse struct {
	UserID string `json:"user_id"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.registerUC.Register(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, emailErrors.ErrInvalidEmail) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, registerResponse{UserID: id})
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	JWT          string `json:"jwt"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	access, refresh, err := h.loginUC.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, loginResponse{JWT: access, RefreshToken: refresh})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type refreshResponse struct {
	JWT          string `json:"jwt"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	access, refresh, err := h.refreshUC.Refresh(req.RefreshToken)
	if err != nil {
		if err == usecase.ErrInvalidRefreshToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, refreshResponse{JWT: access, RefreshToken: refresh})
}
