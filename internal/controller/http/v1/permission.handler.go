package v1

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pharma-crm-backend/config"
	"github.com/pharma-crm-backend/domain"
	"github.com/pharma-crm-backend/pkg/logger"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	cfg *config.Config
	db  *gorm.DB
	log *logger.Logger
}

func NewPermissionHandler(cfg *config.Config, db *gorm.DB, log *logger.Logger) *PermissionHandler {
	return &PermissionHandler{cfg, db, log}
}

func (h *PermissionHandler) Create(c *gin.Context) {
	var body domain.Permission
	var res domain.Permission
	if err := c.ShouldBindJSON(&body); err != nil {
		h.log.Error(err)
		handleResponse(c, BadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	body.Id = uuid.New().String()
	if err := h.db.WithContext(ctx).Model(&domain.Permission{}).Create(&body).Scan(&res).Error; err != nil {
		h.log.Error(err)
		handleResponse(c, InternalError, err.Error())
		return
	}
	handleResponse(c, CREATED, res)
}
