package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisitor_GetSyncHash(t *testing.T) {
	v1 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{3, 1, 2},
	}

	v2 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{1, 2, 3},
	}

	// Test consistency and order independence
	assert.Equal(t, v1.GetSyncHash(), v2.GetSyncHash(), "Hash should be independent of division order")
	assert.NotZero(t, v1.GetSyncHash(), "Hash should not be zero")

	// Test sensitivity to ErpID
	v3 := &Visitor{
		ErpID:          12346,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{1, 2, 3},
	}
	assert.NotEqual(t, v1.GetSyncHash(), v3.GetSyncHash(), "Hash should change when ErpID changes")

	// Test sensitivity to ErpSchoolID
	v4 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S2",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{1, 2, 3},
	}
	assert.NotEqual(t, v1.GetSyncHash(), v4.GetSyncHash(), "Hash should change when ErpSchoolID changes")

	// Test sensitivity to ErpYearGroupID
	v5 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 11,
		ErpDivisions:   []int32{1, 2, 3},
	}
	assert.NotEqual(t, v1.GetSyncHash(), v5.GetSyncHash(), "Hash should change when ErpYearGroupID changes")

	// Test sensitivity to ErpDivisions content
	v6 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{1, 2, 4},
	}
	assert.NotEqual(t, v1.GetSyncHash(), v6.GetSyncHash(), "Hash should change when ErpDivisions content changes")

	// Test sensitivity to ErpDivisions length
	v7 := &Visitor{
		ErpID:          12345,
		ErpSchoolID:    "S1",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{1, 2},
	}
	assert.NotEqual(t, v1.GetSyncHash(), v7.GetSyncHash(), "Hash should change when ErpDivisions length changes")
}

func TestVisitor_IsImportedFromISAMS(t *testing.T) {
	tests := []struct {
		name     string
		visitor  *Visitor
		expected bool
	}{
		{
			name:     "nil visitor",
			visitor:  nil,
			expected: false,
		},
		{
			name: "has ERP id and school id",
			visitor: &Visitor{
				ErpID:       42,
				ErpSchoolID: "S42",
			},
			expected: true,
		},
		{
			name: "missing ERP id",
			visitor: &Visitor{
				ErpSchoolID: "S42",
			},
			expected: false,
		},
		{
			name: "missing school id",
			visitor: &Visitor{
				ErpID: 42,
			},
			expected: false,
		},
		{
			name:     "missing both ERP fields",
			visitor:  &Visitor{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.visitor.IsImportedFromISAMS())
		})
	}
}
