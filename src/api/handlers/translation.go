package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yazu-codes/scanme-translations.git/src/dto"
	"github.com/yazu-codes/scanme-translations.git/src/services"
)

type TranslationHandler struct {
	redisService       *services.RedisService
	translationService *services.TranslationService
	logger             *slog.Logger
}

func NewTranslateHandler(redisService *services.RedisService, translationService *services.TranslationService, logger *slog.Logger) *TranslationHandler {
	return &TranslationHandler{
		redisService:       redisService,
		translationService: translationService,
		logger:             logger,
	}
}

type TranslateRequest struct {
	Menu           dto.PublicMenu `json:"menu" binding:"required"`
	SourceLanguage string         `json:"source_language" binding:"required"`
	TargetLanguage string         `json:"target_language" binding:"required"`
}

// Translate translates a menu to target language
func (h *TranslationHandler) Translate(c *gin.Context) {
	var req TranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Menu.MenuOwner.Name == "" {
		h.logger.Error("Menu owner name is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Menu owner name is required"})
		return
	}

	if len(req.Menu.MenuItems) == 0 {
		h.logger.Error("Menu must have at least one item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Menu must have at least one item"})
		return
	}

	if req.TargetLanguage == "" {
		h.logger.Error("Target language is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target language is required"})
		return
	}

	translatedMenu, err := h.translationService.Translate(req.Menu, req.SourceLanguage, req.TargetLanguage)
	if err != nil {
		h.logger.Error("Translation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Translation failed"})
		return
	}

	h.logger.Info("Translation successful", "target_language", req.TargetLanguage)
	c.JSON(http.StatusOK, translatedMenu)
}
