package service

import (
	"WB-L0/internal/repository"
	"WB-L0/internal/structs"
	"context"
	"log"
)

type Order interface {
	GetOrders(ctx context.Context) ([]structs.Order, error)
	GetOrderByUID(ctx context.Context, orderUID string) (structs.Order, error)
	SaveOrder(ctx context.Context, orderToSave structs.Order) error
	RestoreCache() error
}

type order struct {
	repo repository.Repository
}

func NewOrder(repo repository.Repository) Order {
	return &order{repo: repo}
}

func (o *order) GetOrders(ctx context.Context) ([]structs.Order, error) {
	return o.repo.Order.GetOrders(ctx)
}

func (o *order) GetOrderByUID(ctx context.Context, orduid string) (structs.Order, error) {
	if ord, found := o.repo.Cache.Order.Get(orduid); found {
		return ord, nil
	}

	ord, err := o.repo.Order.GetOrderByUID(ctx, orduid)
	if err != nil {
		return structs.Order{}, err
	}

	o.repo.Cache.Order.Save(orduid, ord)
	return ord, nil
}

func (o *order) SaveOrder(ctx context.Context, ord structs.Order) error {
	if err := o.repo.Order.SaveOrder(ctx, ord); err != nil {
		return err
	}

	o.repo.Cache.Order.Save(ord.OrderUID, ord)
	return nil
}

func (r *order) RestoreCache() error {
	ords, err := r.repo.Order.GetOrders(context.Background())
	if err != nil {
		return err
	}

	for _, order := range ords {
		if success := r.repo.Cache.Order.Save(order.OrderUID, order); !success {
			log.Printf("Error when adding the UID %s order to the cache", order.OrderUID)
			continue
		}
	}

	log.Printf("The cache has been successfully restored from the database. %d ords loaded.", len(ords))
	return nil
}
