package erp

import (
	"context"

	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
)

type Factory interface {
	NewClient(ctx context.Context) (Client, error)
}

type Client interface {
	GetStudents(page, pageSize int32) (*isams.StudentsResponse, error)
}
