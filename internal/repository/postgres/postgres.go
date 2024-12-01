package postgres

import (
	"WB-L0/internal/configs"
	"WB-L0/internal/structs"
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
)

type Order interface {
	GetOrders(ctx context.Context) ([]structs.Order, error)
	GetOrderByUID(ctx context.Context, orderUID string) (structs.Order, error)
	SaveOrder(ctx context.Context, order structs.Order) error
}

func NewPostgresDB(cfg configs.DBConfig) (*sqlx.DB, error) {
	dbStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s password=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.DBName, cfg.SSLMode, cfg.Password)
	db, err := sqlx.Open("postgres", dbStr)
	if err != nil {
		return nil, err
	}

	log.Printf("Connecting to DataBase: host=%s port=%s user=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.DBName, cfg.SSLMode)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

type orderRepository struct {
	ordDB *sqlx.DB
}

func NewOrder(db *sqlx.DB) Order {
	return &orderRepository{ordDB: db}
}

func (o *orderRepository) GetOrders(ctx context.Context) ([]structs.Order, error) {
	var orders []structs.Order
	query := `SELECT * FROM orders`

	if err := o.ordDB.SelectContext(ctx, &orders, query); err != nil {
		return nil, err
	}

	for i, order := range orders {
		var err error
		orders[i].Delivery, orders[i].Payment, orders[i].Items, err = o.GetDataByUID(ctx, order.OrderUID)
		if err != nil {
			return nil, err
		}
	}

	return orders, nil
}

func (o *orderRepository) GetOrderByUID(ctx context.Context, orderUID string) (structs.Order, error) {
	var order structs.Order
	query := `SELECT * FROM orders WHERE uid = $1`

	if err := o.ordDB.GetContext(ctx, &order, query, orderUID); err != nil {
		return structs.Order{}, err
	}

	var err error
	order.Delivery, order.Payment, order.Items, err = o.GetDataByUID(ctx, order.OrderUID)
	if err != nil {
		return structs.Order{}, err
	}

	return order, nil
}

func (o *orderRepository) SaveOrder(ctx context.Context, order structs.Order) error {
	tx, err := o.ordDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	qOrder := `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature, 
            customer_id, delivery_service, shard_key, sm_id, 
            date_created, oof_shard
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
    `
	_, err = tx.ExecContext(ctx, qOrder,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		tx.Rollback()
		return err
	}

	qDelivery := `
        INSERT INTO deliveries (
            order_uid, name, phone, zip, city, address, 
            region, email
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8
        )
    `
	_, err = tx.ExecContext(ctx, qDelivery,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		tx.Rollback()
		return err
	}

	qPayment := `
        INSERT INTO payments (
            order_uid, transaction, request_id, currency, provider, 
            amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
    `
	_, err = tx.ExecContext(ctx, qPayment,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		tx.Rollback()
		return err
	}

	qItem := `
        INSERT INTO items (
            order_uid, chrt_id, track_number, price, rid, name, 
            sale, size, total_price, nm_id, brand, status
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
        )
    `
	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, qItem, order.OrderUID,
			item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale,
			item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (o *orderRepository) GetDataByUID(ctx context.Context, uid string) (delivery structs.Delivery, payment structs.Payment, items []structs.Item, err error) {
	qDelivery := `SELECT * FROM deliveries WHERE order_uid = $1`
	if err = o.ordDB.GetContext(ctx, &delivery, qDelivery, uid); err != nil {
		return
	}

	qPayment := `SELECT * FROM payments WHERE order_uid = $1`
	if err = o.ordDB.GetContext(ctx, &payment, qPayment, uid); err != nil {
		return
	}

	qItems := `SELECT * FROM items WHERE order_uid = $1`
	err = o.ordDB.SelectContext(ctx, &items, qItems, uid)
	return
}
