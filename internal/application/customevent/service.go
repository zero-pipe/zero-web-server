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

// ClearRange 按时间范围删除自定义事件；all=true 时删除全部（忽略范围）。
func (s *Service) ClearRange(_ context.Context, begin, end int64, all bool) (int64, error) {
	if !all {
		if begin <= 0 && end <= 0 {
			return 0, fmt.Errorf("请选择时间范围，或选择全部删除")
		}
		if begin > 0 && end > 0 && begin > end {
			return 0, fmt.Errorf("开始时间不能晚于结束时间")
		}
	}
	return s.repo.DeleteRange(begin, end, all)
}
