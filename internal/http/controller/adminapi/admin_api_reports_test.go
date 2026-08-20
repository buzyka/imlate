package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	usecase "github.com/buzyka/imlate/internal/usecase/adminapi"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupReportsTest() (*providertest.VisitDailyReportRepositoryMock, *AdminAPIController) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	api := &usecase.AdminAPI{VisitDailyReportRepo: mockRepo}
	controller := &AdminAPIController{AdminAPI: api}
	gin.SetMode(gin.TestMode)
	return mockRepo, controller
}

// --- POST /admin-api/reports/visits ---

func TestVisitsReportsPostHandler_Success(t *testing.T) {
	mockRepo, controller := setupReportsTest()

	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{
			Total: 1,
			Rows:  []provider.VisitReportRow{{VisitorID: 1, Name: "John", Surname: "Doe"}},
		}, nil)

	body := usecase.PostReportsVisitsRequest{
		From:   "2026-04-01",
		To:     "2026-04-30",
		Fields: []string{"visitor_id", "name"},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/reports/visits", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.VisitsReportsPostHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp usecase.PostReportsVisitsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Len(t, resp.Data, 1)
	mockRepo.AssertExpectations(t)
}

func TestVisitsReportsPostHandler_MalformedJSON(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/reports/visits", bytes.NewBufferString("{invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.VisitsReportsPostHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsPostHandler_ValidationError(t *testing.T) {
	_, controller := setupReportsTest()

	body := usecase.PostReportsVisitsRequest{
		From:   "bad-date",
		To:     "2026-04-30",
		Fields: []string{"id"},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/reports/visits", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.VisitsReportsPostHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Contains(t, errResp["error"], "invalid 'from' date")
}

func TestVisitsReportsPostHandler_ServerError(t *testing.T) {
	mockRepo, controller := setupReportsTest()

	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database connection lost"))

	body := usecase.PostReportsVisitsRequest{From: "2026-04-01", To: "2026-04-30"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/reports/visits", bytes.NewBuffer(bodyBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.VisitsReportsPostHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// --- GET /admin-api/reports/visits (backward compatibility) ---

func TestVisitsReportsHandler_BackwardCompat(t *testing.T) {
	mockRepo, controller := setupReportsTest()

	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestVisitsReportsHandler_BadIsStudent(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30&is_student=badval", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsHandler_BadYearGroup(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30&year_group=abc", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsHandler_BadPage(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30&page=abc", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsHandler_BadLimit(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30&limit=xyz", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsHandler_UsecaseError(t *testing.T) {
	mockRepo, controller := setupReportsTest()

	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

func TestVisitsReportsHandler_ValidationError(t *testing.T) {
	_, controller := setupReportsTest()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/reports/visits?from=bad-date&to=2026-04-30", nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVisitsReportsHandler_AllValidParams(t *testing.T) {
	mockRepo, controller := setupReportsTest()

	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	url := "/admin-api/reports/visits?from=2026-04-01&to=2026-04-30&is_student=true&year_group=10&sign_status=signed_in&page=2&limit=50"
	c.Request = httptest.NewRequest(http.MethodGet, url, nil)

	controller.VisitsReportsHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}
