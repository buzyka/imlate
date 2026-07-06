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

func boolPtr(b bool) *bool { return &b }

func emptyReportResult() *provider.VisitReportResult {
	return &provider.VisitReportResult{Total: 0, Rows: []provider.VisitReportRow{}}
}

// --- resolveFields ---

func TestResolveFields_Empty_ReturnsAllOptional(t *testing.T) {
	fields, err := resolveFields(nil)
	assert.NoError(t, err)
	// only optional fields returned (computed are always in the response separately)
	assert.Len(t, fields, len(AllowedVisitReportFields)-len(computedFields))
	for _, f := range fields {
		assert.True(t, AllowedVisitReportFields[f])
		assert.False(t, computedFields[f])
	}
}

func TestResolveFields_EmptySlice_ReturnsAllOptional(t *testing.T) {
	fields, err := resolveFields([]string{})
	assert.NoError(t, err)
	assert.Len(t, fields, len(AllowedVisitReportFields)-len(computedFields))
}

func TestResolveFields_ValidSubset(t *testing.T) {
	fields, err := resolveFields([]string{"visitor_id", "name"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"visitor_id", "name"}, fields)
}

func TestResolveFields_ComputedFieldAllowed(t *testing.T) {
	// passing a computed field name in fields is allowed (not an error)
	fields, err := resolveFields([]string{"visit_date", "name"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"visit_date", "name"}, fields)
}

func TestResolveFields_UnknownField_ReturnsError(t *testing.T) {
	_, err := resolveFields([]string{"visitor_id", "unknown_field"})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "unknown_field")
}

// --- buildPostRowMap ---

func TestBuildPostRowMap_ComputedFieldsAlwaysPresent(t *testing.T) {
	signedIn := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	signedOut := time.Date(2026, 3, 10, 16, 0, 0, 0, time.UTC)
	dur := 480
	row := provider.VisitReportRow{
		VisitDate:       time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		SignStatus:      "signed_out",
		VisitsCount:     2,
		SignedIn:        &signedIn,
		SignedOut:       &signedOut,
		DurationMinutes: &dur,
	}

	// request only an optional field — computed fields must still appear
	result := buildPostRowMap(row, []string{"name"})

	assert.Equal(t, "2026-03-10", result["visit_date"])
	assert.Equal(t, "signed_out", result["sign_status"])
	assert.Equal(t, 2, result["visits_count"])
	assert.Equal(t, &signedIn, result["signed_in"])
	assert.Equal(t, &signedOut, result["signed_out"])
	assert.Equal(t, &dur, result["duration_minutes"])
}

func TestBuildPostRowMap_ComputedFieldsWithNilPointers(t *testing.T) {
	row := provider.VisitReportRow{
		VisitDate:       time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		SignStatus:      "not_signed",
		VisitsCount:     0,
		SignedIn:        nil,
		SignedOut:       nil,
		DurationMinutes: nil,
	}

	result := buildPostRowMap(row, []string{})

	assert.Equal(t, "2026-03-10", result["visit_date"])
	assert.Nil(t, result["signed_in"])
	assert.Nil(t, result["signed_out"])
	assert.Nil(t, result["duration_minutes"])
}

func TestBuildPostRowMap_SelectedOptionalFields(t *testing.T) {
	yg := 10
	row := provider.VisitReportRow{
		VisitorID: 42,
		Name:      "John",
		Surname:   "Smith",
		Email:     "john@test.com",
		Image:     "/img/john.jpg",
		YearGroup: &yg,
	}

	result := buildPostRowMap(row, []string{"visitor_id", "name", "email"})

	// optional fields: only requested ones
	assert.Equal(t, 42, result["visitor_id"])
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, "john@test.com", result["email"])
	_, hasSurname := result["surname"]
	assert.False(t, hasSurname)
	_, hasImage := result["image"]
	assert.False(t, hasImage)
	// computed fields always present
	_, hasVisitDate := result["visit_date"]
	assert.True(t, hasVisitDate)
}

func TestBuildPostRowMap_AllOptionalFields(t *testing.T) {
	yg := 11
	signedIn := time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)
	row := provider.VisitReportRow{
		VisitorID:   99,
		VisitDate:   time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Name:        "Alice",
		Surname:     "Wonder",
		Email:       "alice@test.com",
		Image:       "/img/alice.jpg",
		IsStudent:   true,
		YearGroup:   &yg,
		VisitsCount: 1,
		SignStatus:  "signed_in",
		SignedIn:    &signedIn,
	}

	result := buildPostRowMap(row, []string{"visitor_id", "name", "surname", "is_student", "year_group", "email", "image"})

	assert.Len(t, result, len(AllowedVisitReportFields)) // 6 computed + 7 optional = 13
	assert.Equal(t, 99, result["visitor_id"])
	assert.Equal(t, "Alice", result["name"])
	assert.Equal(t, "Wonder", result["surname"])
	assert.Equal(t, true, result["is_student"])
	assert.Equal(t, &yg, result["year_group"])
	assert.Equal(t, "alice@test.com", result["email"])
	assert.Equal(t, "/img/alice.jpg", result["image"])
	assert.Equal(t, "2026-03-10", result["visit_date"])
}

func TestBuildPostRowMap_NilYearGroup(t *testing.T) {
	row := provider.VisitReportRow{VisitorID: 1, YearGroup: nil}
	result := buildPostRowMap(row, []string{"year_group"})
	assert.Nil(t, result["year_group"])
}

func TestBuildPostRowMap_ComputedFieldPassedInFields_NotDuplicated(t *testing.T) {
	row := provider.VisitReportRow{
		VisitDate:  time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		SignStatus: "not_signed",
	}
	// pass a computed field explicitly — should not cause duplication or error
	result := buildPostRowMap(row, []string{"visit_date", "name"})
	assert.Equal(t, "2026-03-10", result["visit_date"])
	// still only one entry for visit_date
	assert.Len(t, result, len(computedFields)+1) // 6 computed + name
}

// --- GetPostReportsVisits validation ---

func TestGetPostReportsVisits_InvalidFromDate(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{From: "bad-date", To: "2026-04-30"})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "invalid 'from' date")
}

func TestGetPostReportsVisits_InvalidToDate(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{From: "2026-04-01", To: "bad-date"})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "invalid 'to' date")
}

func TestGetPostReportsVisits_UnknownField(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:   "2026-04-01",
		To:     "2026-04-30",
		Fields: []string{"visitor_id", "unknown"},
	})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "unknown")
}

func TestGetPostReportsVisits_InvalidSignStatus(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01",
		To:   "2026-04-30",
		Filters: &PostReportsVisitsFilters{
			SignStatus: []string{"signed_in", "bad_status"},
		},
	})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "bad_status")
}

func TestGetPostReportsVisits_InvalidSortField(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:  "2026-04-01",
		To:    "2026-04-30",
		Order: &PostReportsVisitsOrder{Field: "unknown_field", Direction: "asc"},
	})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "unknown_field")
}

func TestGetPostReportsVisits_InvalidSortDirection(t *testing.T) {
	api := &AdminAPI{}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:  "2026-04-01",
		To:    "2026-04-30",
		Order: &PostReportsVisitsOrder{Field: "sign_status", Direction: "sideways"},
	})
	assert.ErrorIs(t, err, ErrInvalidRequestFormat)
	assert.Contains(t, err.Error(), "sideways")
}

func TestGetPostReportsVisits_PageDefaultsTo1(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.Page == 1 && f.PageSize == DefaultReportsVisitsPageSize
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{From: "2026-04-01", To: "2026-04-30"})
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, DefaultReportsVisitsPageSize, resp.Limit)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_PageAndLimitFromRequest(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.Page == 3 && f.PageSize == 25
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01", To: "2026-04-30",
		Page: 3, Limit: 25,
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, resp.Page)
	assert.Equal(t, 25, resp.Limit)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_RepositoryError(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db failure"))

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{From: "2026-04-01", To: "2026-04-30"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get visit report")
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_EmptyData(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{From: "2026-04-01", To: "2026-04-30"})
	assert.NoError(t, err)
	assert.Empty(t, resp.Data)
	assert.Equal(t, 0, resp.Total)
	assert.Equal(t, 0, resp.TotalPages)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_TotalPagesCalculation(t *testing.T) {
	tests := []struct {
		total         int
		limit         int
		expectedPages int
	}{
		{40, 20, 2},
		{41, 20, 3},
		{5, 20, 1},
		{0, 20, 0},
		{100, 15, 7},
	}

	for _, tc := range tests {
		mockRepo := new(providertest.VisitDailyReportRepositoryMock)
		mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
			Return(&provider.VisitReportResult{Total: tc.total, Rows: []provider.VisitReportRow{}}, nil)

		api := &AdminAPI{VisitDailyReportRepo: mockRepo}
		resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
			From: "2026-04-01", To: "2026-04-30",
			Limit: tc.limit,
		})
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedPages, resp.TotalPages)
		mockRepo.AssertExpectations(t)
	}
}

func TestGetPostReportsVisits_MultiValueSignStatus(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return len(f.SignStatuses) == 2 &&
				f.SignStatuses[0] == "signed_in" &&
				f.SignStatuses[1] == "signed_out"
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01",
		To:   "2026-04-30",
		Filters: &PostReportsVisitsFilters{
			SignStatus: []string{"signed_in", "signed_out"},
		},
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_MultiValueYearGroup(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return len(f.YearGroups) == 3 &&
				f.YearGroups[0] == 10 &&
				f.YearGroups[1] == 11 &&
				f.YearGroups[2] == 12
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01",
		To:   "2026-04-30",
		Filters: &PostReportsVisitsFilters{
			YearGroup: []int{10, 11, 12},
		},
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_IsStudentFilter(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.IsStudent != nil && *f.IsStudent == true
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01",
		To:   "2026-04-30",
		Filters: &PostReportsVisitsFilters{
			IsStudent: boolPtr(true),
		},
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_SortFieldAndDirection(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.OrderField == "year_group" && f.OrderDirection == "asc"
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:  "2026-04-01",
		To:    "2026-04-30",
		Order: &PostReportsVisitsOrder{Field: "year_group", Direction: "asc"},
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_SortDesc(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.OrderField == "sign_status" && f.OrderDirection == "desc"
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:  "2026-04-01",
		To:    "2026-04-30",
		Order: &PostReportsVisitsOrder{Field: "sign_status", Direction: "desc"},
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_OrderNil_NoOrderInFilter(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.OrderField == "" && f.OrderDirection == ""
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01", To: "2026-04-30",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_FieldSelection_ComputedAlwaysPresent(t *testing.T) {
	visitDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{
			Total: 1,
			Rows: []provider.VisitReportRow{
				{
					VisitorID:   42,
					VisitDate:   visitDate,
					Name:        "John",
					Surname:     "Smith",
					SignStatus:  "signed_in",
					VisitsCount: 1,
					Email:       "john@test.com",
				},
			},
		}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From:   "2026-04-01",
		To:     "2026-04-30",
		Fields: []string{"visitor_id", "name"},
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	item := resp.Data[0]

	// requested optional fields
	assert.Equal(t, 42, item["visitor_id"])
	assert.Equal(t, "John", item["name"])

	// computed fields always present
	assert.Equal(t, "2026-03-10", item["visit_date"])
	assert.Equal(t, "signed_in", item["sign_status"])
	assert.Equal(t, 1, item["visits_count"])

	// non-requested optional fields absent
	_, hasSurname := item["surname"]
	assert.False(t, hasSurname)
	_, hasEmail := item["email"]
	assert.False(t, hasEmail)

	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_EmptyFields_ReturnsAllFields(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
		Return(&provider.VisitReportResult{
			Total: 1,
			Rows:  []provider.VisitReportRow{{VisitorID: 1, Name: "A", Surname: "B", Email: "a@b.com"}},
		}, nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	resp, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01", To: "2026-04-30",
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	// 6 computed + 7 optional = 13
	assert.Len(t, resp.Data[0], len(AllowedVisitReportFields))
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_DateSwap(t *testing.T) {
	expectedFrom := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	expectedToExclusive := time.Date(2026, 4, 31, 0, 0, 0, 0, time.UTC)

	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", expectedFrom, expectedToExclusive, mock.Anything).
		Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-30",
		To:   "2026-04-01",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_NilFilters_NoFilterInRepoCall(t *testing.T) {
	mockRepo := new(providertest.VisitDailyReportRepositoryMock)
	mockRepo.On("GetVisitReport", mock.Anything, mock.Anything,
		mock.MatchedBy(func(f provider.VisitReportFilter) bool {
			return f.IsStudent == nil &&
				len(f.SignStatuses) == 0 &&
				len(f.YearGroups) == 0
		}),
	).Return(emptyReportResult(), nil)

	api := &AdminAPI{VisitDailyReportRepo: mockRepo}
	_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
		From: "2026-04-01", To: "2026-04-30",
	})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetPostReportsVisits_AllValidSortFields(t *testing.T) {
	for field := range allowedSortFields {
		mockRepo := new(providertest.VisitDailyReportRepositoryMock)
		mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
			Return(emptyReportResult(), nil)

		api := &AdminAPI{VisitDailyReportRepo: mockRepo}
		_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
			From:  "2026-04-01",
			To:    "2026-04-30",
			Order: &PostReportsVisitsOrder{Field: field, Direction: "asc"},
		})
		assert.NoError(t, err, "field %s should be valid", field)
		mockRepo.AssertExpectations(t)
	}
}

func TestGetPostReportsVisits_AllValidSignStatuses(t *testing.T) {
	for status := range validSignStatuses {
		mockRepo := new(providertest.VisitDailyReportRepositoryMock)
		mockRepo.On("GetVisitReport", mock.Anything, mock.Anything, mock.Anything).
			Return(emptyReportResult(), nil)

		api := &AdminAPI{VisitDailyReportRepo: mockRepo}
		_, err := api.GetPostReportsVisits(&PostReportsVisitsRequest{
			From: "2026-04-01",
			To:   "2026-04-30",
			Filters: &PostReportsVisitsFilters{
				SignStatus: []string{status},
			},
		})
		assert.NoError(t, err, "status %s should be valid", status)
		mockRepo.AssertExpectations(t)
	}
}
