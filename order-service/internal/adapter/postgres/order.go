package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/19parwiz/order-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepo struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *OrderRepo {
	return &OrderRepo{pool: pool}
}

func (o *OrderRepo) Create(ctx context.Context, order domain.Order) error {
	tx, err := o.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`INSERT INTO orders (id, user_id, total_amount, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		int64(order.ID),
		int64(order.UserID),
		order.TotalAmount,
		string(order.Status),
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, total_price)
			 VALUES ($1, $2, $3, $4)`,
			int64(order.ID),
			int64(item.ProductID),
			int64(item.Quantity),
			item.TotalPrice,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	return nil
}

func (o *OrderRepo) Update(ctx context.Context, filter domain.OrderFilter, update domain.OrderUpdateData) error {
	setClauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	argPos := 1

	if update.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, string(*update.Status))
		argPos++
	}
	if update.UpdatedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argPos))
		args = append(args, *update.UpdatedAt)
		argPos++
	}
	if len(setClauses) == 0 {
		return nil
	}

	where, whereArgs, nextPos := orderFilterToWhere(filter, argPos)
	if where == "" {
		return domain.ErrOrderNotFound
	}
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE orders SET %s WHERE %s", strings.Join(setClauses, ", "), where)
	tag, err := o.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	_ = nextPos
	return nil
}

func (o *OrderRepo) GetWithFilter(ctx context.Context, filter domain.OrderFilter) (domain.Order, error) {
	where, args, _ := orderFilterToWhere(filter, 1)
	if where == "" {
		return domain.Order{}, domain.ErrOrderNotFound
	}

	query := fmt.Sprintf(`
SELECT id, user_id, total_amount, status, created_at, updated_at
FROM orders
WHERE %s
LIMIT 1`, where)

	var order domain.Order
	var id, userID int64
	var status string
	err := o.pool.QueryRow(ctx, query, args...).Scan(
		&id,
		&userID,
		&order.TotalAmount,
		&status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, domain.ErrOrderNotFound
		}
		return domain.Order{}, fmt.Errorf("failed to get order: %w", err)
	}
	order.ID = uint64(id)
	order.UserID = uint64(userID)
	order.Status = domain.OrderStatus(status)

	items, err := o.getOrderItems(ctx, order.ID)
	if err != nil {
		return domain.Order{}, err
	}
	order.Items = items
	return order, nil
}

func (o *OrderRepo) GetAllWithFilter(ctx context.Context, filter domain.OrderFilter, page, limit int64) ([]domain.Order, int64, error) {
	where, args, nextPos := orderFilterToWhere(filter, 1)
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	countQuery := "SELECT COUNT(*) FROM orders"
	if where != "" {
		countQuery += " WHERE " + where
	}
	var total int64
	if err := o.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	listQuery := `SELECT id, user_id, total_amount, status, created_at, updated_at FROM orders`
	if where != "" {
		listQuery += " WHERE " + where
	}
	listQuery += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", nextPos, nextPos+1)
	listArgs := append(args, limit, offset)

	rows, err := o.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var (
			order        domain.Order
			id, userID   int64
			statusString string
		)
		if err := rows.Scan(&id, &userID, &order.TotalAmount, &statusString, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan order row: %w", err)
		}
		order.ID = uint64(id)
		order.UserID = uint64(userID)
		order.Status = domain.OrderStatus(statusString)
		items, err := o.getOrderItems(ctx, order.ID)
		if err != nil {
			return nil, 0, err
		}
		order.Items = items
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate order rows: %w", err)
	}
	return orders, total, nil
}

func (o *OrderRepo) Delete(ctx context.Context, filter domain.OrderFilter) error {
	where, args, _ := orderFilterToWhere(filter, 1)
	if where == "" {
		return domain.ErrOrderNotFound
	}

	query := fmt.Sprintf("DELETE FROM orders WHERE %s", where)
	tag, err := o.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (o *OrderRepo) getOrderItems(ctx context.Context, orderID uint64) ([]domain.OrderItem, error) {
	rows, err := o.pool.Query(
		ctx,
		`SELECT product_id, quantity, total_price FROM order_items WHERE order_id = $1`,
		int64(orderID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var productID, quantity int64
		var item domain.OrderItem
		if err := rows.Scan(&productID, &quantity, &item.TotalPrice); err != nil {
			return nil, fmt.Errorf("failed to scan order item row: %w", err)
		}
		item.ProductID = uint64(productID)
		item.Quantity = uint64(quantity)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate order item rows: %w", err)
	}

	return items, nil
}

func orderFilterToWhere(filter domain.OrderFilter, argStart int) (string, []any, int) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 3)
	argPos := argStart

	if filter.ID != nil {
		clauses = append(clauses, fmt.Sprintf("id = $%d", argPos))
		args = append(args, int64(*filter.ID))
		argPos++
	}
	if filter.UserID != nil {
		clauses = append(clauses, fmt.Sprintf("user_id = $%d", argPos))
		args = append(args, int64(*filter.UserID))
		argPos++
	}
	if filter.Status != nil {
		clauses = append(clauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, string(*filter.Status))
		argPos++
	}

	return strings.Join(clauses, " AND "), args, argPos
}
