// Package firelist serves the public, server-rendered evacuation roster page.
// It is intentionally outside the admin SPA and outside JWT auth: during an
// evacuation the page has to open on a phone in seconds.
package firelist

import (
	"errors"
	"net/http"

	firelistview "github.com/buzyka/imlate/internal/usecase/firelist"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// fireListTemplateName is the template registered on the gin engine at startup.
const fireListTemplateName = "firelist.html"

const (
	invalidGradeMessage = "Unknown class. Check the link and try again."
	// unavailableMessage is deliberately generic: the underlying error may name
	// tables or connection strings, and this page is public.
	unavailableMessage = "Class data is unavailable right now. Reload the page or use the paper register."
)

type FireListController struct {
	FireList *firelistview.Service `container:"type"`
	Logger   *zap.SugaredLogger    `container:"type"`
}

// FireListPageHandler renders the roster for one class.
func (c *FireListController) FireListPageHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// The roster changes minute by minute during an evacuation; a cached copy
		// would show a child as still inside after they were signed out.
		ctx.Header("Cache-Control", "no-store")

		grade := ctx.Param("grade")

		data, err := c.FireList.GetFireList(grade)
		if err != nil {
			if errors.Is(err, firelistview.ErrAlarmInactive) {
				// 403 rather than 200: the roster is withheld, and anything
				// watching this endpoint — a log, a monitor, a scraper — should
				// be able to tell that apart from a successful read.
				ctx.HTML(http.StatusForbidden, fireListTemplateName,
					firelistview.InactivePageData(grade))
				return
			}
			if errors.Is(err, firelistview.ErrInvalidGrade) {
				// A readable page, not a JSON error: whoever opened this link is
				// standing outside with a class.
				ctx.HTML(http.StatusNotFound, fireListTemplateName,
					firelistview.ErrorPageData(grade, invalidGradeMessage))
				return
			}
			c.logf("firelist: failed to build list for grade %q: %v", grade, err)
			ctx.HTML(http.StatusInternalServerError, fireListTemplateName,
				firelistview.ErrorPageData(grade, unavailableMessage))
			return
		}

		ctx.HTML(http.StatusOK, fireListTemplateName, data)
	}
}

func (c *FireListController) logf(format string, args ...interface{}) {
	if c.Logger == nil {
		return
	}
	c.Logger.Errorf(format, args...)
}
