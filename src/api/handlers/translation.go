package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

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

	key := req.Menu.MenuOwner.Name + "_" + req.TargetLanguage

	// TODO: If there is req.Menu.MenuOwner.Name+"_"+req.TargetLanguage in redis, pull the information from there and serve it and return
	exists, err := h.redisService.Exists(c, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis translation failed"})
		return
	}

	if exists {
		h.logger.Info("Menu Translation Cache Record Found!")
		cached, err := h.redisService.Get(c, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis pull failed"})
			return
		}
		if cached != "" {
			var cachedMenu dto.PublicMenu
			if err := json.Unmarshal([]byte(cached), &cachedMenu); err == nil {
				c.JSON(http.StatusOK, cachedMenu)
				return
			}
		}
	}

	translatedMenu, err := h.translationService.Translate(req.Menu, req.SourceLanguage, req.TargetLanguage)
	if err != nil {
		h.logger.Error("Translation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Translation failed"})
		return
	}

	// TODO: Save translated menu as req.Menu.MenuOwner.Name+"_"+req.TargetLanguage in redis
	// Save translated menu as req.Menu.MenuOwner.Name+"_"+req.TargetLanguage in redis
	data, err := json.Marshal(translatedMenu)
	if err != nil {
		h.logger.Error("Failed to marshal translated menu for caching", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Translation failed: Failed to marshal translated menu for caching"})
	} else {
		if err := h.redisService.Set(c.Request.Context(), key, string(data), 7*24*time.Hour); err != nil {
			h.logger.Error("Failed to cache translated menu", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Save to cache failed"})
		}
	}

	h.logger.Info("Translation successful", "target_language", req.TargetLanguage)
	c.JSON(http.StatusOK, translatedMenu)
}
