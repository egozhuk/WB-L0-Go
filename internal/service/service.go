package service

import "WB-L0/internal/repository"

type Service struct {
	Order
}

func NewService(repositories repository.Repository) Service {
	return Service{Order: NewOrder(repositories)}
}
