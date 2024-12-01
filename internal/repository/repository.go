package repository

import (
	"WB-L0/internal/repository/cache"
	"WB-L0/internal/repository/postgres"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	Order postgres.Order
	Cache cache.Cache
}

func NewRepository(db *sqlx.DB) Repository {
	return Repository{
		Order: postgres.NewOrder(db),
		Cache: cache.NewCache(),
	}
}
