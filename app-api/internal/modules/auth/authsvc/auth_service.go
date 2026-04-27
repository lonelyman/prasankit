package authsvc

import "prasankit-api/internal/modules/auth"

type Service struct {
	repository auth.Repository
}

func NewService(repository auth.Repository) *Service {
	return &Service{
		repository: repository,
	}
}
