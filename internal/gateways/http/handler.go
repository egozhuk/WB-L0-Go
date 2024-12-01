package http

import (
	"WB-L0/internal/gateways/models"
	"WB-L0/internal/service"
	"WB-L0/internal/structs"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
)

type Order interface {
	GetOrders(c *gin.Context)
	GetOrderByUID(c *gin.Context)
	SaveOrder(c *gin.Context)
}

type Handler struct {
	Order
}

func NewHandler(service service.Service) *Handler {
	handler := newOrder(service.Order)
	return &Handler{Order: handler}
}

type order struct {
	service  service.Order
	validate *validator.Validate
}

func newOrder(service service.Order) Order {
	return &order{
		service:  service,
		validate: validator.New(),
	}
}

func (o *order) GetOrders(c *gin.Context) {
	orders, err := o.service.GetOrders(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve orders",
		})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No orders found",
		})
		return
	}

	responseOrders := checkArr(orders)
	c.JSON(http.StatusOK, responseOrders)
}

func (o *order) GetOrderByUID(c *gin.Context) {
	orderUID := c.Param("order_uid")
	if orderUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order UID is required",
		})
		return
	}

	orderRes, err := o.service.GetOrderByUID(c, orderUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Failed to get order " + orderUID,
		})
		return
	}

	responseOrder := checkDomain(orderRes)
	c.JSON(http.StatusOK, responseOrder)
}

func (o *order) SaveOrder(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("The panic has been restored: %v\n", r)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}()

	var order structs.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		fmt.Printf("JSON binding error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := o.validate.Struct(order); err != nil {
		fmt.Printf("Validation error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed"})
		return
	}

	fmt.Printf("Order has been received for saving: %+v\n", order)

	if err := o.service.SaveOrder(c, order); err != nil {
		fmt.Printf("Error saving the order: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order saved successfully"})
}

func checkArr(orders []structs.Order) []models.Order {
	res := make([]models.Order, len(orders))
	for i, order := range orders {
		res[i] = checkDomain(order)
	}
	return res
}

func checkDomain(order structs.Order) models.Order {
	return models.Order{
		OrderUID:          order.OrderUID,
		TrackNumber:       order.TrackNumber,
		Entry:             order.Entry,
		Delivery:          checkDelivery(order.Delivery),
		Payment:           checkPayment(order.Payment),
		Items:             checkItems(order.Items),
		Locale:            order.Locale,
		InternalSignature: order.InternalSignature,
		CustomerID:        order.CustomerID,
		DeliveryService:   order.DeliveryService,
		Shardkey:          order.ShardKey,
		SmID:              order.SmID,
		DateCreated:       order.DateCreated,
		OofShard:          order.OofShard,
	}
}

func checkDelivery(d structs.Delivery) models.Delivery {
	return models.Delivery{
		Name:    d.Name,
		Phone:   d.Phone,
		ZIP:     d.Zip,
		City:    d.City,
		Address: d.Address,
		Region:  d.Region,
		Email:   d.Email,
	}
}

func checkPayment(p structs.Payment) models.Payment {
	return models.Payment{
		Transaction:  p.Transaction,
		RequestID:    p.RequestID,
		Currency:     p.Currency,
		Provider:     p.Provider,
		Amount:       p.Amount,
		PaymentDT:    p.PaymentDT,
		Bank:         p.Bank,
		DeliveryCost: p.DeliveryCost,
		GoodsTotal:   p.GoodsTotal,
		CustomFee:    p.CustomFee,
	}
}

func checkItems(items []structs.Item) []models.Item {
	convertedItems := make([]models.Item, len(items))
	for i, item := range items {
		convertedItems[i] = models.Item{
			ChrtID:      item.ChrtID,
			TrackNumber: item.TrackNumber,
			Price:       item.Price,
			RID:         item.Rid,
			Name:        item.Name,
			Sale:        item.Sale,
			Size:        item.Size,
			TotalPrice:  item.TotalPrice,
			NmID:        item.NmID,
			Brand:       item.Brand,
			Status:      item.Status,
		}
	}
	return convertedItems
}
