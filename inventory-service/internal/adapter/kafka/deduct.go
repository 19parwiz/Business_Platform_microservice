package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/19parwiz/inventory-service/internal/domain"
	events "github.com/19parwiz/inventory-service/protos/gen/golang"
)

var (
	// ErrOrderLineInvalid means the line item is nil or has no product or quantity.
	ErrOrderLineInvalid = errors.New("kafka: invalid order line item")
	// ErrInsufficientStock means on-hand stock is below the ordered quantity.
	ErrInsufficientStock = errors.New("kafka: insufficient stock")
)

// ProductStock is the subset of product operations needed to apply order line deductions.
type ProductStock interface {
	Get(ctx context.Context, pf domain.ProductFilter) (domain.Product, error)
	Update(ctx context.Context, filter domain.ProductFilter, updated domain.ProductUpdateData) error
}

// ProcessOrderLineItem loads stock, checks quantity, and persists the new stock level.
// It returns ErrOrderLineInvalid for nil or empty ids/qty, ErrInsufficientStock when stock is too low,
// or an error wrapping Get/Update failures.
func ProcessOrderLineItem(ctx context.Context, uc ProductStock, item *events.OrderItemEvent) error {
	if item == nil {
		return ErrOrderLineInvalid
	}
	productID := item.GetProductId()
	qty := item.GetQuantity()
	if productID == 0 || qty == 0 {
		return ErrOrderLineInvalid
	}

	filter := domain.ProductFilter{ID: &productID}
	current, err := uc.Get(ctx, filter)
	if err != nil {
		return fmt.Errorf("get product_id=%d: %w", productID, err)
	}
	if current.Stock < qty {
		return fmt.Errorf("%w for product_id=%d: have %d need %d", ErrInsufficientStock, productID, current.Stock, qty)
	}

	newStock := current.Stock - qty
	upd := domain.ProductUpdateData{Stock: &newStock}
	if err := uc.Update(ctx, filter, upd); err != nil {
		return fmt.Errorf("update product_id=%d: %w", productID, err)
	}
	return nil
}
