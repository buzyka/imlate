package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/stretchr/testify/assert"
)

func TestFindByKey_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	expectedVisitor := &entity.Visitor{
		Id:      1,
		Name:    "John",
		Surname: "Doe",
		Grade:   10,
		Image:   "/assets/img/teachers/1.jpg",
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(1, "John", "Doe", 10, "/assets/img/teachers/1.jpg", "ABC123")

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("ABC123").
		WillReturnRows(rows)

	// Execute
	result, err := repo.FindByKey("abc123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Visitor)
	assert.Equal(t, expectedVisitor.Id, result.Visitor.Id)
	assert.Equal(t, expectedVisitor.Name, result.Visitor.Name)
	assert.Equal(t, expectedVisitor.Surname, result.Visitor.Surname)
	assert.Equal(t, expectedVisitor.Grade, result.Visitor.Grade)
	assert.Equal(t, expectedVisitor.Image, result.Visitor.Image)
	assert.Equal(t, "ABC123", result.Key)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByKey_NotFound(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("NOTFOUND").
		WillReturnError(sql.ErrNoRows)

	// Execute
	result, err := repo.FindByKey("notfound")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Visitor)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByKey_DatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	expectedError := errors.New("database connection error")
	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("ERROR").
		WillReturnError(expectedError)

	// Execute
	result, err := repo.FindByKey("error")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByKey_NullGrade(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(1, "John", "Doe", nil, "/assets/img/teachers/1.jpg", "KEY123")

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("KEY123").
		WillReturnRows(rows)

	// Execute
	result, err := repo.FindByKey("key123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Visitor)
	assert.Equal(t, 0, result.Visitor.Grade)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByKey_NullImage(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(1, "John", "Doe", 10, nil, "KEY123")

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("KEY123").
		WillReturnRows(rows)

	// Execute
	result, err := repo.FindByKey("key123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Visitor)
	// Image should be set by AddRandomImage
	assert.NotEmpty(t, result.Visitor.Image)
	assert.Contains(t, result.Visitor.Image, "/assets/img/teachers/")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindById_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image"}).
		AddRow(123, "Jane", "Smith", 11, "/assets/img/teachers/2.jpg")

	mock.ExpectQuery("SELECT id, name, surname, grade, image FROM visitors WHERE id = ?").
		WithArgs(int32(123)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.FindById(123)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(123), result.Id)
	assert.Equal(t, "Jane", result.Name)
	assert.Equal(t, "Smith", result.Surname)
	assert.Equal(t, 11, result.Grade)
	assert.Equal(t, "/assets/img/teachers/2.jpg", result.Image)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindById_NotFound(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	mock.ExpectQuery("SELECT id, name, surname, grade, image FROM visitors WHERE id = ?").
		WithArgs(int32(999)).
		WillReturnError(sql.ErrNoRows)

	// Execute
	result, err := repo.FindById(999)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(0), result.Id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindById_DatabaseError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	expectedError := errors.New("connection lost")
	mock.ExpectQuery("SELECT id, name, surname, grade, image FROM visitors WHERE id = ?").
		WithArgs(int32(123)).
		WillReturnError(expectedError)

	// Execute
	result, err := repo.FindById(123)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindById_NullValues(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image"}).
		AddRow(456, "Bob", "Johnson", nil, nil)

	mock.ExpectQuery("SELECT id, name, surname, grade, image FROM visitors WHERE id = ?").
		WithArgs(int32(456)).
		WillReturnRows(rows)

	// Execute
	result, err := repo.FindById(456)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(456), result.Id)
	assert.Equal(t, 0, result.Grade)
	assert.NotEmpty(t, result.Image)
	assert.Contains(t, result.Image, "/assets/img/teachers/")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddKeyToVisitor_Success(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      100,
		Name:    "Alice",
		Surname: "Brown",
	}

	// Expect FindByKey to return empty result (key not assigned)
	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("NEWKEY").
		WillReturnError(sql.ErrNoRows)

	// Expect INSERT query
	mock.ExpectExec("INSERT INTO visitor_key").
		WithArgs(int32(100), "NEWKEY").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Execute
	err = repo.AddKeyToVisitor(visitor, "NEWKEY")

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddKeyToVisitor_SameVisitor(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      100,
		Name:    "Alice",
		Surname: "Brown",
	}

	// Key already assigned to the same visitor
	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(100, "Alice", "Brown", 10, "/test.jpg", "EXISTKEY")

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("EXISTKEY").
		WillReturnRows(rows)

	// Execute
	err = repo.AddKeyToVisitor(visitor, "EXISTKEY")

	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddKeyToVisitor_DifferentVisitor(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      100,
		Name:    "Alice",
		Surname: "Brown",
	}

	// Key already assigned to different visitor
	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(200, "Charlie", "Davis", 9, "/test.jpg", "TAKEN")

	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("TAKEN").
		WillReturnRows(rows)

	// Execute
	err = repo.AddKeyToVisitor(visitor, "TAKEN")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Key already assigned to another visitor")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddKeyToVisitor_SearchError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      100,
		Name:    "Alice",
		Surname: "Brown",
	}

	expectedError := errors.New("database error")
	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("ERRORKEY").
		WillReturnError(expectedError)

	// Execute
	err = repo.AddKeyToVisitor(visitor, "ERRORKEY")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Search by key error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddKeyToVisitor_InsertError(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      100,
		Name:    "Alice",
		Surname: "Brown",
	}

	// Key not assigned
	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("NEWKEY").
		WillReturnError(sql.ErrNoRows)

	expectedError := errors.New("insert failed")
	mock.ExpectExec("INSERT INTO visitor_key").
		WithArgs(int32(100), "NEWKEY").
		WillReturnError(expectedError)

	// Execute
	err = repo.AddKeyToVisitor(visitor, "NEWKEY")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddRandomImage(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor := &entity.Visitor{
		Id:      1,
		Name:    "Test",
		Surname: "User",
	}

	// Execute
	repo.AddRandomImage(visitor)

	// Assert
	assert.NotEmpty(t, visitor.Image)
	assert.Contains(t, visitor.Image, "/assets/img/teachers/")
	assert.Contains(t, visitor.Image, ".jpg")
}

func TestAddRandomImage_MultipleCalls(t *testing.T) {
	// Setup
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	visitor1 := &entity.Visitor{Id: 1, Name: "User1", Surname: "Test1"}
	visitor2 := &entity.Visitor{Id: 2, Name: "User2", Surname: "Test2"}

	// Execute
	repo.AddRandomImage(visitor1)
	repo.AddRandomImage(visitor2)

	// Assert
	assert.NotEmpty(t, visitor1.Image)
	assert.NotEmpty(t, visitor2.Image)
	assert.Contains(t, visitor1.Image, "/assets/img/teachers/")
	assert.Contains(t, visitor2.Image, "/assets/img/teachers/")

	// Extract numbers from paths
	var num1, num2 int
	fmt.Sscanf(visitor1.Image, "/assets/img/teachers/%d.jpg", &num1)
	fmt.Sscanf(visitor2.Image, "/assets/img/teachers/%d.jpg", &num2)

	assert.True(t, num1 >= 1 && num1 <= 11, "Image number should be between 1 and 11")
	assert.True(t, num2 >= 1 && num2 <= 11, "Image number should be between 1 and 11")
}

func TestFindByKey_CaseInsensitive(t *testing.T) {
	// Setup
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{
		Connection: db,
	}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "grade", "image", "key_id"}).
		AddRow(1, "John", "Doe", 10, "/assets/img/teachers/1.jpg", "MIXEDCASE")

	// The query should receive uppercase version
	mock.ExpectQuery("SELECT v.id, v.name, v.surname, v.grade, v.image, vk.key_id FROM visitors AS v INNER JOIN visitor_key AS vk").
		WithArgs("MIXEDCASE").
		WillReturnRows(rows)

	// Execute with lowercase
	result, err := repo.FindByKey("MixedCase")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Visitor)
	assert.Equal(t, "MIXEDCASE", result.Key)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAll_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "name", "surname", "is_student", "grade", "image", "isams_id", "isams_school_id", "updated_at"}).
		AddRow(1, "John", "Doe", true, 10, "img.jpg", 1001, "S1001", now).
		AddRow(2, "Jane", "Doe", false, nil, nil, nil, nil, nil)

	mock.ExpectQuery("SELECT id, name, surname, is_student, grade, image, isams_id, isams_school_id, updated_at FROM visitors ORDER BY id ASC").
		WillReturnRows(rows)

	visitors, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, visitors, 2)

	assert.Equal(t, int32(1), visitors[0].Id)
	assert.Equal(t, "John", visitors[0].Name)
	assert.Equal(t, 10, visitors[0].Grade)
	assert.Equal(t, "img.jpg", visitors[0].Image)
	assert.Equal(t, int64(1001), visitors[0].ErpID)
	assert.Equal(t, "S1001", visitors[0].ErpSchoolID)
	assert.Equal(t, now, visitors[0].UpdatedAt)

	assert.Equal(t, int32(2), visitors[1].Id)
	assert.Equal(t, "Jane", visitors[1].Name)
	assert.Equal(t, 0, visitors[1].Grade)
	assert.NotEmpty(t, visitors[1].Image) // Random image should be assigned
	assert.Equal(t, int64(0), visitors[1].ErpID)
	assert.Equal(t, "", visitors[1].ErpSchoolID)
}

func TestGetAll_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	mock.ExpectQuery("SELECT id, name, surname, is_student, grade, image, isams_id, isams_school_id, updated_at FROM visitors ORDER BY id ASC").
		WillReturnError(errors.New("query error"))

	visitors, err := repo.GetAll()
	assert.Error(t, err)
	assert.Nil(t, visitors)
}

func TestGetAll_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	rows := sqlmock.NewRows([]string{"id", "name", "surname", "is_student", "grade", "image", "isams_id", "isams_school_id", "updated_at"}).
		AddRow("invalid", "John", "Doe", true, 10, "img.jpg", 1001, "S1001", time.Now())

	mock.ExpectQuery("SELECT id, name, surname, is_student, grade, image, isams_id, isams_school_id, updated_at FROM visitors ORDER BY id ASC").
		WillReturnRows(rows)

	visitors, err := repo.GetAll()
	assert.Error(t, err)
	assert.Nil(t, visitors)
}

func TestAddVisitor_Insert_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	now := time.Now()
	visitor := &entity.Visitor{
		Name:        "John",
		Surname:     "Doe",
		IsStudent:   true,
		Grade:       10,
		Image:       "img.jpg",
		ErpID:       1001,
		ErpSchoolID: "S1001",
		UpdatedAt:   now,
	}

	mock.ExpectExec("INSERT INTO visitors").
		WithArgs("John", "Doe", true, 10, "img.jpg", 1001, "S1001", now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddVisitor(visitor)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), visitor.Id)
}

func TestAddVisitor_Insert_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	visitor := &entity.Visitor{Name: "John"}

	mock.ExpectExec("INSERT INTO visitors").
		WillReturnError(errors.New("insert error"))

	err = repo.AddVisitor(visitor)
	assert.Error(t, err)
}

func TestAddVisitor_Insert_LastInsertIdError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	visitor := &entity.Visitor{Name: "John"}

	mock.ExpectExec("INSERT INTO visitors").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))

	err = repo.AddVisitor(visitor)
	assert.Error(t, err)
}

func TestAddVisitor_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	now := time.Now()
	visitor := &entity.Visitor{
		Id:          1,
		Name:        "John",
		Surname:     "Doe",
		IsStudent:   true,
		Grade:       10,
		Image:       "img.jpg",
		ErpID:       1001,
		ErpSchoolID: "S1001",
		UpdatedAt:   now,
	}

	mock.ExpectExec("UPDATE visitors").
		WithArgs("John", "Doe", true, 10, "img.jpg", 1001, "S1001", now, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddVisitor(visitor)
	assert.NoError(t, err)
}

func TestAddVisitor_Update_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	visitor := &entity.Visitor{Id: 1, Name: "John"}

	mock.ExpectExec("UPDATE visitors").
		WillReturnError(errors.New("update error"))

	err = repo.AddVisitor(visitor)
	assert.Error(t, err)
}

func TestAddVisitor_Update_NullFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	visitor := &entity.Visitor{
		Id:          1,
		Name:        "John",
		Surname:     "Doe",
		IsStudent:   false,
		Grade:       0,
		Image:       "",
		ErpID:       0,
		ErpSchoolID: "",
		UpdatedAt:   time.Time{},
	}

	mock.ExpectExec("UPDATE visitors").
		WithArgs("John", "Doe", false, 0, "", sql.NullInt64{}, sql.NullString{}, time.Time{}, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddVisitor(visitor)
	assert.NoError(t, err)
}

func TestAddVisitor_Insert_NullFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := &Visitor{Connection: db}

	visitor := &entity.Visitor{
		Name:        "John",
		Surname:     "Doe",
		IsStudent:   false,
		Grade:       0,
		Image:       "",
		ErpID:       0,
		ErpSchoolID: "",
		UpdatedAt:   time.Time{},
	}

	mock.ExpectExec("INSERT INTO visitors").
		WithArgs("John", "Doe", false, 0, "", sql.NullInt64{}, sql.NullString{}, time.Time{}).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddVisitor(visitor)
	assert.NoError(t, err)
}
