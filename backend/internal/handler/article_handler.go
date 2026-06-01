package handler

import (
	"net/http"
	"strconv"

	"loledgeagent/internal/repository"
	"loledgeagent/internal/service"

	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	repo     *repository.ArticleRepo
	fetchSvc *service.FetchService
}

func NewArticleHandler(repo *repository.ArticleRepo, fetchSvc *service.FetchService) *ArticleHandler {
	return &ArticleHandler{repo: repo, fetchSvc: fetchSvc}
}

func (h *ArticleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sourceID, _ := strconv.Atoi(c.DefaultQuery("source_id", "0"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	list, total, err := h.repo.List(uint(sourceID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
		"items": list, "total": total, "page": page, "page_size": limit,
	}})
}

func (h *ArticleHandler) FetchAll(c *gin.Context) {
	articles, err := h.fetchSvc.FetchAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": gin.H{
		"articles_fetched": len(articles),
	}})
}
