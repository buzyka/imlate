package entity

import (
	"encoding/binary"
	"hash/crc64"
	"sort"
	"time"
)

// CRC table (ISO)
var crcTable = crc64.MakeTable(crc64.ISO)

type Visitor struct {
	Id      int32  `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
	IsStudent bool `json:"is_student"`
	Grade   int    `json:"grade"`	
	Image   string `json:"image"`
	ErpID   int64 `json:"isams_id"`
	ErpSchoolID string `json:"isams_school_id"`
	ErpYearGroupID int32 `json:"isams_year_group_id"`
	ErpDivisions []int32 `json:"isams_divisions"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VisitDetails struct {
	Visitor  *Visitor `json:"visitor"`
	Key 	 string   `json:"key"`
}

func (v *Visitor) GetSyncHash() uint64 {
	divs := make([]int32, len(v.ErpDivisions))
	copy(divs, v.ErpDivisions)
	sort.Slice(divs, func(i, j int) bool { return divs[i] < divs[j] })

	h := crc64.New(crcTable)

	// Write ErpID (int64)
	_ = binary.Write(h, binary.BigEndian, v.ErpID)

	// Separator to avoid ambiguity when concatenating bytes
	h.Write([]byte{0x00})

	// Write ErpSchoolID (string)
	h.Write([]byte(v.ErpSchoolID))
	h.Write([]byte{0x00})

	// Write ErpYearGroupID (int32)
	_ = binary.Write(h, binary.BigEndian, v.ErpYearGroupID)
	h.Write([]byte{0x00})

	// Write ErpDivisions ([]int32)
	for _, div := range divs {
		_ = binary.Write(h, binary.BigEndian, div)
	}

	return h.Sum64()
}
