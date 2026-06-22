package mocks

import (
	"context"

	"github.com/m-bromo/go-auth-template/internal/repository"
)

type UnitOfWork struct {
	ExecFunc func(ctx context.Context, fn func(repos repository.Repositories) error) error

	ExecCalls int
	Repos     repository.Repositories
}

func (m *UnitOfWork) Exec(ctx context.Context, fn func(repos repository.Repositories) error) error {
	m.ExecCalls++

	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, fn)
	}

	return fn(m.Repos)
}
