package adminapi

import (
	"errors"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetReportsVisits_ValidateRequestError(t *testing.T) {
	testCase := []struct {
		name     string
		from     string
		to       string
		opts     []ReportsVisitsOption
		exErrMsg string
	}{
		{
			name:     "missing 'to' date",
			from:     "2026-01-01",
			opts:     []ReportsVisitsOption{},
			exErrMsg: "invalid 'to' date",
		},
		{
			name:     "invalid 'from' date",
			from:     "invalid-date",
			to:       "2026-01-02",
			opts:     []ReportsVisitsOption{},
			exErrMsg: "invalid 'from' date",
		},
		{
			name:     "invalid 'sign_status' value",
			from:     "2026-01-01",
			to:       "2026-01-02",
			opts:     []ReportsVisitsOption{WithSignStatus("invalid_status")},
			exErrMsg: "invalid 'sign_status' value",
		},
		{
			name:     "invalid 'page' value",
			from:     "2026-01-01",
			to:       "2026-01-02",
			opts:     []ReportsVisitsOption{WithPage(0)},
			exErrMsg: "'page' must be a positive integer",
		},
		{
			name:     "year_group filter used when is_student is false",
			from:     "2026-01-01",
			to:       "2026-01-02",
			opts:     []ReportsVisitsOption{WithIsStudent(false), WithYearGroup(10)},
			exErrMsg: "'year_group' filter can only be used when 'is_student' is true",
		},
		{
			name:     "limit above the maximum page size",
			from:     "2026-01-01",
			to:       "2026-01-02",
			opts:     []ReportsVisitsOption{WithLimit(MaxReportsVisitsPageSize + 1)},
			exErrMsg: "'limit' must not exceed",
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			api := &AdminAPI{}
			_, err := api.GetReportsVisits(tc.from, tc.to, tc.opts...)
			assert.ErrorIs(t, err, ErrInvalidRequestFormat)
			assert.Contains(t, err.Error(), tc.exErrMsg)
		})
	}
}

func TestGetReportsVisits_Success(t *testing.T) {
	signedIn := time.Date(2026, 3, 10, 8, 45, 12, 0, time.UTC)
	signedOut := time.Date(2026, 3, 10, 15, 12, 55, 0, time.UTC)
	yg := 10
	dur := 387

	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{
			Total: 1,
			Rows: []provider.VisitReportRow{
				{
					VisitDate:       time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
					VisitorID:       25,
					Name:            "John",
					Surname:         "Smith",
					IsStudent:       true,
					YearGroup:       &yg,
					VisitsCount:     2,
					SignStatus:      "signed_out",
					SignedIn:        &signedIn,
					SignedOut:       &signedOut,
					DurationMinutes: &dur,
				},
			},
		}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, DefaultReportsVisitsPageSize, resp.Limit)
	assert.Equal(t, 1, resp.TotalPages)
	assert.Len(t, resp.Data, 1)

	item := resp.Data[0]
	assert.Equal(t, "2026-03-10", item.VisitDate)
	assert.Equal(t, 25, item.VisitorID)
	assert.Equal(t, "John", item.Name)
	assert.Equal(t, "Smith", item.Surname)
	assert.True(t, item.IsStudent)
	assert.NotNil(t, item.YearGroup)
	assert.Equal(t, 10, *item.YearGroup)
	assert.Equal(t, 2, item.VisitsCount)
	assert.Equal(t, "signed_out", item.SignStatus)
	assert.NotNil(t, item.SignedIn)
	assert.NotNil(t, item.SignedOut)
	assert.NotNil(t, item.DurationMinutes)
	assert.Equal(t, 387, *item.DurationMinutes)
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_RepositoryError(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("database connection lost"))

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to get visit report")
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_TotalPagesCalculation(t *testing.T) {
	tests := []struct {
		name          string
		total         int
		pageSize      int
		expectedPages int
	}{
		{"exact division", 40, 20, 2},
		{"with remainder", 41, 20, 3},
		{"single page", 5, 20, 1},
		{"zero records", 0, 20, 0},
		{"custom page size", 100, 15, 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(providertest.VisitDailyReportRepositoryMock)
			mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
				Return(&provider.VisitReportResult{
					Total: tc.total,
					Rows:  []provider.VisitReportRow{},
				}, nil)

			api := &AdminAPI{VisitDailyReportRepo: mockRepo}
			opts := []ReportsVisitsOption{WithLimit(tc.pageSize)}
			resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14", opts...)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedPages, resp.TotalPages)
			assert.Equal(t, tc.total, resp.Total)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetReportsVisits_DateSwap(t *testing.T) {
	expectedFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	expectedToExclusive := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", expectedFrom, expectedToExclusive, mock.Anything).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-14", "2026-03-01")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_FiltersPassedToRepository(t *testing.T) {
	isStudent := true
	expectedFilter := provider.VisitReportFilter{
		IsStudent:    &isStudent,
		YearGroups:   []int{10},
		SignStatuses: []string{"signed_in"},
		Page:         2,
		PageSize:     15,
	}

	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, expectedFilter).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	opts := []ReportsVisitsOption{
		WithIsStudent(true),
		WithYearGroup(10),
		WithSignStatus("signed_in"),
		WithPage(2),
		WithLimit(15),
	}
	_, err := api.GetReportsVisits("2026-03-01", "2026-03-14", opts...)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_EmptyData(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Total)
	assert.Empty(t, resp.Data)
	assert.Equal(t, 0, resp.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_NullableFieldsMapping(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{
			Total: 1,
			Rows: []provider.VisitReportRow{
				{
					VisitDate:       time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
					VisitorID:       50,
					Name:            "Test",
					Surname:         "User",
					IsStudent:       false,
					YearGroup:       nil,
					VisitsCount:     0,
					SignStatus:      "not_signed",
					SignedIn:        nil,
					SignedOut:       nil,
					DurationMinutes: nil,
				},
			},
		}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14")

	assert.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	item := resp.Data[0]
	assert.Nil(t, item.YearGroup)
	assert.Nil(t, item.SignedIn)
	assert.Nil(t, item.SignedOut)
	assert.Nil(t, item.DurationMinutes)
	assert.Equal(t, "not_signed", item.SignStatus)
	mockRepo.AssertExpectations(t)
}

func TestGetReportsVisits_DefaultPagination(t *testing.T) {
	expectedFilter := provider.VisitReportFilter{
		Page:         1,
		PageSize:     DefaultReportsVisitsPageSize,
		YearGroups:   nil,
		SignStatuses: nil,
	}

	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, expectedFilter).
		Return(&provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetReportsVisits("2026-03-01", "2026-03-14")

	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, DefaultReportsVisitsPageSize, resp.Limit)
	mockRepo.AssertExpectations(t)
}
