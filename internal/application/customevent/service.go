package customevent

import (
	"context"
	"fmt"

	domaince "zero-web-server/internal/domain/customevent"
)

// Service 提供自定义事件的查询（写入由事件入库链路直接落库）。
type Service struct {
	repo domaince.Repository
}

func NewService(repo domaince.Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(_ context.Context, page, count int, f domaince.ListFilter) ([]*domaince.CustomEvent, int64, error) {
	return s.repo.List(page, count, f)
}

func (s *Service) Get(_ context.Context, id string) (*domaince.CustomEvent, error) {
	e, err := s.repo.Get(id)
	if err != nil {
		return nil, fmt.Errorf("自定义事件不存在")
	}
	return e, nil
}

func (s *Service) Types(_ context.Context, deviceInternalCode string) ([]domaince.TypeCount, error) {
	return s.repo.DistinctTypes(deviceInternalCode)
}
