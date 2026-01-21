package synchroniser

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"go.uber.org/zap"
)

const PageSize = 100

type StudentSync struct {
	Config				*config.Config             `container:"type"`
	ERPFactory         erp.Factory                 `container:"type"`
	VisitorRepo        provider.VisitorRepository  `container:"type"`
	Logger 			   *zap.SugaredLogger		   `container:"type"`	
	currentVisitors    []*entity.Visitor
	ctx                context.Context
	currentClient      erp.Client
	yearGroupDivisions map[int32][]int32
}

func (s *StudentSync) cleanUpSyncSession() {
	s.currentClient = nil
	s.ctx = nil
	s.yearGroupDivisions = nil
}

func (s *StudentSync) startSyncSession(ctx context.Context) error {
	ERPClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		return err
	}
	s.currentClient = ERPClient
	return nil
}

func (s *StudentSync) SyncAllStudents() error {
	ctx := context.Background()
	err := s.startSyncSession(ctx)
	if err != nil {
		s.Logger.Errorw("sync all students: create ERP client failed", "error", err)
		return err
	}
	defer s.cleanUpSyncSession()
	s.yearGroupDivisions = make(map[int32][]int32)

	s.currentVisitors, err = s.VisitorRepo.GetAll()
	if err != nil {
		s.Logger.Errorw("sync all students: get all visitors failed", "error", err)
		return err
	}

	var pageNumber int32 = 1

	for {
		s.Logger.Infof("Syncing students, page %d", pageNumber)
		resp, err := s.currentClient.GetStudents(pageNumber, PageSize)
		if err != nil {
			s.Logger.Errorw("sync all students: get students step failed", "pageNumber", pageNumber,  "pageSize", PageSize, "error", err)
			return err
		}

		for _, student := range resp.Students {
			if student.ID == 5225 {
				fmt.Printf("----- STUDENT: %s\n", *student.Forename)
			}
			if err = s.SaveStudent(student); err != nil {
				s.Logger.Errorw("sync all students: save student step failed", "studentID", student.ID, "schoolID", student.SchoolID, "error", err)
				return err
			}
		}
		if resp.TotalPages <= pageNumber {
			break
		}
		pageNumber++
	}
	return nil
}

func (s *StudentSync) SyncRegistrationCodesDictionaries() error {
	ctx := context.Background()
	err := s.startSyncSession(ctx)
	if err != nil {
		return err
	}
	defer s.cleanUpSyncSession()

	codesDict := &entity.RegistrationCodeDictionary{
		Codes: make(map[int32]*entity.RegistrationCode),
		UploadedAt: time.Now().Truncate(time.Second),
	}


	// Absence codes
	absenceCodesResp, err := s.currentClient.GetRegistrationAbsenceCodes()
	if err != nil {
		return err
	}
	for _, code := range absenceCodesResp.AbsenceCodes {
		codesDict.Codes[code.ID] = &entity.RegistrationCode{
			ID:            code.ID,
			Code:		   code.Code,
			Name:          code.Name,
			IsAbsenceCode: true,
		}
	}
	entity.SetAbsenceCodeDictionary(codesDict)

	codesDict = &entity.RegistrationCodeDictionary{
		Codes: make(map[int32]*entity.RegistrationCode),
		UploadedAt: time.Now().Truncate(time.Second),
	}

	// Present codes
	presentCodesResp, err := s.currentClient.GetRegistrationPresentCodes()
	if err != nil {
		return err
	}
	for _, code := range presentCodesResp.PresentCodes {
		codesDict.Codes[code.ID] = &entity.RegistrationCode{
			ID:            code.ID,
			Code:		   code.Code,
			Name:          code.Name,
			IsAbsenceCode: false,
		}
	}

	entity.SetPresentsCodeDictionary(codesDict)
	return nil	
}

func (s *StudentSync) SyncStudentPhotos() error {
	ctx := context.Background()
	err := s.startSyncSession(ctx)
	if err != nil {
		return err
	}
	defer s.cleanUpSyncSession()

	visitors, err := s.VisitorRepo.GetAll()
	if err != nil {
		return err
	}
	for _, visitor := range visitors {
		if !visitor.IsStudent {
			continue
		}
		photoResp, err := s.currentClient.GetStudentPhoto(visitor.ErpSchoolID)
		if err != nil {
			if strings.Contains(err.Error(), "Not Found") {
				continue
			}
			return err
		}
		if photoResp.Data == nil {
			continue
		}
		filePath := s.Config.StudentsImagePhotoDir
		fileName := visitor.ErpSchoolID + photoResp.Extension
		if err := os.WriteFile(filePath + "/" + fileName, photoResp.Data, 0644); err != nil {
			return err
		} else {
			visitor.Image = s.Config.StudentsImagePhotoURLPrefix + "/" + fileName
			if err := s.VisitorRepo.AddVisitor(visitor); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *StudentSync) SaveStudent(student isams.Student) error {
	var fullName, forename, surname string

	if student.Surname != nil && *student.Surname != "" && student.Forename != nil && *student.Forename != "" {
		surname = *student.Surname
		forename = *student.Forename
		fullName = forename + " " + surname
	}

	if student.FullName != nil && *student.FullName != "" {
		fullName = *student.FullName
	}

	var grade, yearGroup int
	if student.YearGroup != nil {
		grade = *student.YearGroup
		yearGroup = *student.YearGroup
	}

	divisions, err := s.getDivisionsByYearGroup(int32(yearGroup))
	if err != nil {
		s.Logger.Errorw("get divisions by year group failed", "yearGroupID", yearGroup, "error", err)
		return err
	}

	var UpdatedAt time.Time
	if student.LastUpdated != nil {
		parsedTime, err := time.Parse(time.RFC3339, *student.LastUpdated)
		if err == nil {
			UpdatedAt = parsedTime.Truncate(time.Second)
		}
	}

	visitor := &entity.Visitor{
		Name:           forename,
		Surname:        surname,
		FullName:	    fullName,
		IsStudent:      true,
		Grade:          grade,
		ErpID:          student.ID,
		ErpSchoolID:    student.SchoolID,
		ErpYearGroupID: int32(yearGroup),
		ErpDivisions:   divisions,
		UpdatedAt:      UpdatedAt,
	}

	if (visitor.ErpID == 5225) {
		fmt.Printf("--SYNC: %s %s\n", visitor.Surname, visitor.Name)
		fmt.Printf("Current: %d", visitor.GetSyncHash())
	}

	// Check if visitor needs to be updated or added
	// and set visitor.Id if exists
	if !s.IsUpToDate(visitor) {
		s.Logger.Infof("Saving student visitor: ERP ID %d, Name: %s", visitor.ErpID, visitor.FullName)
		return s.VisitorRepo.SaveVisitor(visitor)
	}
	return nil
}

func (s *StudentSync) IsUpToDate(newVisitor *entity.Visitor) bool {
	for _, currentVisitor := range s.currentVisitors {
		if currentVisitor.ErpID == newVisitor.ErpID {
			
			if (newVisitor.ErpID == 5225) {
				fmt.Printf("+++ Existing: %d\n", currentVisitor.GetSyncHash())
				fmt.Printf("+++ New: %d\n", newVisitor.GetSyncHash())
				fmt.Printf("++++ Is up to date: %v\n", currentVisitor.GetSyncHash() == newVisitor.GetSyncHash())
				fmt.Printf("++++ UpdatedAt equal: %v\n", currentVisitor.UpdatedAt.Equal(newVisitor.UpdatedAt))
			}
			
			newVisitor.Id = currentVisitor.Id
			if currentVisitor.GetSyncHash() == newVisitor.GetSyncHash() {
				return true
			}
		}
	}
	if newVisitor.UpdatedAt.IsZero() {
		newVisitor.UpdatedAt = time.Now().Truncate(time.Second)
	}
	return false
}

func (s *StudentSync) getDivisionsByYearGroup(erpID int32) ([]int32, error) {
	if divisions, ok := s.yearGroupDivisions[erpID]; ok {
		return divisions, nil
	}

	resp, err := s.currentClient.GetYearGroupDivisions(erpID)
	if err != nil {
		return nil, err
	}
	var divisions []int32
	for _, division := range resp.Divisions {
		divisions = append(divisions, division.ID)
	}
	s.yearGroupDivisions[erpID] = divisions
	return divisions, nil
}
