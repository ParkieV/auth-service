package rest

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/ParkieV/auth-service/internal/domain"
	"github.com/ParkieV/auth-service/internal/usecase"
)

type Handler struct {
	registerUC *usecase.RegisterUsecase
	loginUC    *usecase.LoginUsecase
	refreshUC  *usecase.RefreshUsecase
	logoutUC   *usecase.LogoutUsecase
	verifyUC   *usecase.VerifyUsecase
}

func RegisterHandlers(
	r *gin.Engine,
	registerUC *usecase.RegisterUsecase,
	loginUC *usecase.LoginUsecase,
	refreshUC *usecase.RefreshUsecase,
	logoutUC *usecase.LogoutUsecase,
	verifyUC *usecase.VerifyUsecase,
) {
	h := &Handler{registerUC, loginUC, refreshUC, logoutUC, verifyUC}

	api := r.Group("/api")
	{
		api.POST("/register", h.register)
		api.POST("/login", h.login)
		api.POST("/refresh", h.refresh)
		api.POST("/logout", h.logout)
		api.POST("/verify", h.verify)
	}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
type registerResponse struct {
	UserID string `json:"user_id"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.registerUC.Register(c.Request.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, registerResponse{UserID: id})
	case errors.Is(err, domain.ErrInvalidEmail):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrEmailExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
type loginResponse struct {
	JWT          string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	at, rt, err := h.loginUC.Login(c.Request.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, loginResponse{JWT: at, RefreshToken: rt})
	case errors.Is(err, usecase.ErrNotConfirmed):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrUserNotFound),
		errors.Is(err, usecase.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
type refreshResponse struct {
	JWT          string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	at, rt, err := h.refreshUC.Refresh(c.Request.Context(), req.RefreshToken)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, refreshResponse{JWT: at, RefreshToken: rt})
	case errors.Is(err, usecase.ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	UserID       string `json:"user_id" binding:"required"`
}

func (h *Handler) logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.logoutUC.Logout(c.Request.Context(), req.UserID, req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Status(http.StatusNoContent)
}

type verifyRequest struct {
	Token string `json:"access_token" binding:"required"`
}
type verifyResponse struct {
	UserID string `json:"user_id"`
}

func (h *Handler) verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.verifyUC.Verify(c.Request.Context(), req.Token)
	switch {
	case err == nil && res.Active:
		c.JSON(http.StatusOK, verifyResponse{UserID: res.UserID})
	case errors.Is(err, usecase.ErrTokenInvalid):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
