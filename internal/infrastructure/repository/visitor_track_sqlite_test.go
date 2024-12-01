package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	_ "github.com/go-sql-driver/mysql"
)

func TestGetDailyStat(t *testing.T) {
	vt := &VisitorTrack{getConnection(t)}
	// now := time.Now()
	// sevenDaysAgo := now.AddDate(0, 0, -7)	
	_, err := vt.GetDailyStat(time.Now())
	assert.Nil(t, err)
}

func getConnection(t *testing.T) *sql.DB {
	dbSource := "trackme:trackme@tcp(localhost:3307)/tracker"
	db, err := sql.Open("mysql", dbSource)
	assert.Nil(t, err)	
	return db
}