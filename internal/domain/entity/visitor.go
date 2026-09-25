package entity

import (
	"encoding/binary"
	"fmt"
	"hash/crc64"
	"sort"
	"strings"
	"time"
)

// CRC table (ISO)
var crcTable = crc64.MakeTable(crc64.ISO)

// FormGroupMaxLength is the size of the visitors.form_group column.
const FormGroupMaxLength = 32

type Visitor struct {
	Id             int32      `json:"id"`
	Name           string     `json:"name"`
	Surname        string     `json:"surname"`
	FullName       string     `json:"full_name"`
	Email          string     `json:"email"`
	IsStudent      bool       `json:"is_student"`
	Grade          *int       `json:"grade"`
	Image          string     `json:"image"`
	ErpID          int64      `json:"isams_id"`
	ErpSchoolID    string     `json:"isams_school_id"`
	ErpYearGroupID int32      `json:"isams_year_group_id"`
	FormGroup      *string    `json:"form_group"`
	ErpDivisions   []int32    `json:"isams_divisions"`
	SyncHash       uint64     `json:"-"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Keys           []string   `json:"keys"`
}

type VisitDetails struct {
	Visitor *Visitor `json:"visitor"`
	Key     string   `json:"key"`
}

func (v *Visitor) IsImportedFromISAMS() bool {
	if v == nil {
		return false
	}

	return v.ErpID != 0 && v.ErpSchoolID != ""
}

// NormalizeFormGroup maps a form group to what is stored: nil and blank values
// become nil, anything else is trimmed at the edges only ("4 B" stays).
func NormalizeFormGroup(formGroup *string) *string {
	if formGroup == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*formGroup)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (v *Visitor) GetSyncHash() uint64 {
	if v.SyncHash == 0 {
		v.SyncHash = v.calcSyncHash()
	}
	return v.SyncHash
}
func (v *Visitor) calcSyncHash() uint64 {
	divs := make([]int32, len(v.ErpDivisions))
	copy(divs, v.ErpDivisions)
	sort.Slice(divs, func(i, j int) bool { return divs[i] < divs[j] })

	h := crc64.New(crcTable)

	// Write ErpID (int64)
	_ = binary.Write(h, binary.BigEndian, v.ErpID)

	// Write FullName (string)
	_, _ = fmt.Fprintf(h, "%s %s", v.Name, v.Surname)

	// Separator to avoid ambiguity when concatenating bytes
	h.Write([]byte{0x00})

	// Write ErpSchoolID (string)
	h.Write([]byte(v.ErpSchoolID))
	h.Write([]byte{0x00})

	// Write ErpYearGroupID (int32)
	_ = binary.Write(h, binary.BigEndian, v.ErpYearGroupID)
	h.Write([]byte{0x00})

	// Write FormGroup (*string); a presence marker keeps nil apart from ""
	if v.FormGroup == nil {
		h.Write([]byte{0x00})
	} else {
		h.Write([]byte{0x01})
		h.Write([]byte(*v.FormGroup))
	}
	h.Write([]byte{0x00})

	// Write ErpDivisions ([]int32)
	for _, div := range divs {
		_ = binary.Write(h, binary.BigEndian, div)
	}

	return h.Sum64()
}
