package visitor

import (
	"net/http"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/gin-gonic/gin"
)

type AddKeyRequest struct {
	VisitorID int32 `json:"visitor_id"`
	VisitorKey  string `json:"visitor_key"`
}

type VisitorController struct {
	VisitorRepository provider.VisitorRepository `container:"type"`
}

func (vc *VisitorController) AddKeyHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var Request AddKeyRequest
		err := ctx.Bind(&Request)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		visitor, err := vc.VisitorRepository.FindById(Request.VisitorID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		if visitor == nil || visitor.Id == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Visitor not exists",
			})
			return
		}

		if err = vc.VisitorRepository.AddKeyToVisitor(visitor, Request.VisitorKey); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"message": "Key successfully added",
		})
	}
}
