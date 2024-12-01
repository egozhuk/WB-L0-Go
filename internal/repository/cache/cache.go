package cache

type Cache struct {
	Order Order
}

func NewCache() Cache {
	return Cache{Order: newOrder()}
}
