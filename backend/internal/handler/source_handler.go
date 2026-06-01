package handler

import (
	"net/http"
	"strconv"

	"loledgeagent/internal/models"
	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
)

type SourceHandler struct {
	repo     *repository.SourceRepo
	fetchSvc *service.FetchService
}

func NewSourceHandler(repo *repository.SourceRepo, fetchSvc *service.FetchService) *SourceHandler {
	return &SourceHandler{repo: repo, fetchSvc: fetchSvc}
}

func (h *SourceHandler) List(c *gin.Context) {
	enabledOnly := c.Query("enabled") == "true"
	list, err := h.repo.List(enabledOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": list})
}

func (h *SourceHandler) Create(c *gin.Context) {
	var s models.Source
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.repo.Create(&s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "success", "data": s})
}

func (h *SourceHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	s, err := h.repo.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "source not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": s})
}

func (h *SourceHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	s, err := h.repo.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "source not found"})
		return
	}
	if err := c.ShouldBindJSON(s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	s.ID = uint(id)
	if err := h.repo.Update(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": s})
}

func (h *SourceHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.repo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

func (h *SourceHandler) Fetch(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	count, err := h.fetchSvc.FetchSource(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{"articles_fetched": count}})
}
