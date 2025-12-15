package search

import (
	"net/http"

	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/gin-gonic/gin"
)

type SearchController struct {
	StudentRepository entity.VisitorRepository `container:"type"`
}

func (sc SearchController) SearchHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.Param("id")
		visit, err := sc.StudentRepository.FindByKey(id)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		if visit == nil || visit.Visitor == nil {
			ctx.Status(http.StatusNotFound)
			return
		}
		ctx.JSON(http.StatusOK, visit)
	}
}
