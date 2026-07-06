package adminapi

import (
	"errors"
	"net/http"
	"strconv"

	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
)

// ReportsVisitsResponse is a Swagger-visible alias for the use case response.
type ReportsVisitsResponse = usecase.ReportsVisitsResponse

// VisitsReportsHandler godoc
// @Summary      Get visitor attendance report
// @Description  Returns paginated visitor attendance information aggregated per visitor per day for a given date range.
// @Tags         admin-reports
// @Produce      json
// @Param        from        query     string  true   "Start date (YYYY-MM-DD, inclusive)"
// @Param        to          query     string  true   "End date (YYYY-MM-DD, inclusive)"
// @Param        is_student  query     boolean false  "Filter by student flag"
// @Param        year_group  query     integer false  "Filter by year group"
// @Param        sign_status query     string  false  "Filter by sign status (not_signed, signed_in, signed_out)"
// @Param        page        query     integer false  "Page number (default 1)"
// @Param        limit       query     integer false  "Records per page (default 100)"
// @Success      200  {object}  ReportsVisitsResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Deprecated   true
// @Router       /admin-api/reports/visits [get]
func (ac *AdminAPIController) VisitsReportsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		from := c.Query("from")
		to := c.Query("to")

		var opts []usecase.ReportsVisitsOption

		if v := c.Query("is_student"); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'is_student' value, must be boolean"})
				return
			}
			opts = append(opts, usecase.WithIsStudent(b))
		}

		if v := c.Query("year_group"); v != "" {
			yg, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'year_group' value, must be integer"})
				return
			}
			opts = append(opts, usecase.WithYearGroup(yg))
		}

		if v := c.Query("sign_status"); v != "" {
			opts = append(opts, usecase.WithSignStatus(v))
		}

		if v := c.Query("page"); v != "" {
			p, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'page' value, must be integer"})
				return
			}
			opts = append(opts, usecase.WithPage(p))
		}

		if v := c.Query("limit"); v != "" {
			l, err := strconv.Atoi(v)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'limit' value, must be integer"})
				return
			}
			opts = append(opts, usecase.WithLimit(l))
		}

		resp, err := ac.AdminAPI.GetReportsVisits(from, to, opts...)
		if err != nil {
			if errors.Is(err, usecase.ErrInvalidRequestFormat) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// VisitsReportsPostHandler godoc
// @Summary      Search visitor attendance report
// @Description  Returns a paginated, field-selectable visitor attendance report with multi-value filters and server-side sorting.
// @Tags         admin-reports
// @Accept       json
// @Produce      json
// @Param        body  body      PostReportsVisitsRequest  true  "Search criteria"
// @Success      200   {object}  PostReportsVisitsResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/reports/visits [post]
func (ac *AdminAPIController) VisitsReportsPostHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req usecase.PostReportsVisitsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}

		resp, err := ac.AdminAPI.GetPostReportsVisits(&req)
		if err != nil {
			if errors.Is(err, usecase.ErrInvalidRequestFormat) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
