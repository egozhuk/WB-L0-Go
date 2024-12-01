package cache

import (
	"WB-L0/internal/structs"
	"sync"
)

type Order interface {
	Get(uid string) (structs.Order, bool)
	Save(uid string, orderToSave structs.Order) bool
}

type order struct {
	orders map[string]structs.Order
	mu     sync.RWMutex
}

func newOrder() Order {
	return &order{orders: make(map[string]structs.Order)}
}

func (o *order) Get(uid string) (structs.Order, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	order, exists := o.orders[uid]
	return order, exists
}

func (o *order) Save(uid string, orderToSave structs.Order) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.orders[uid] = orderToSave
	return true
}
