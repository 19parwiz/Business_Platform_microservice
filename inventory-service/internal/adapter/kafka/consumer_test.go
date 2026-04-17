package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/19parwiz/inventory-service/internal/domain"
	events "github.com/19parwiz/inventory-service/protos/gen/golang"
)

type stubStock struct {
	product   domain.Product
	getErr    error
	updateErr error
	getN      int
	updateN   int
	lastStock uint64
}

func (s *stubStock) Get(ctx context.Context, pf domain.ProductFilter) (domain.Product, error) {
	s.getN++
	return s.product, s.getErr
}

func (s *stubStock) Update(ctx context.Context, filter domain.ProductFilter, updated domain.ProductUpdateData) error {
	s.updateN++
	if updated.Stock != nil {
		s.lastStock = *updated.Stock
	}
	return s.updateErr
}

func TestProcessOrderLineItem_nilItem(t *testing.T) {
	var s stubStock
	err := ProcessOrderLineItem(context.Background(), &s, nil)
	if !errors.Is(err, ErrOrderLineInvalid) {
		t.Fatalf("got %v want ErrOrderLineInvalid", err)
	}
	if s.getN != 0 || s.updateN != 0 {
		t.Fatalf("getN=%d updateN=%d want 0,0", s.getN, s.updateN)
	}
}

func TestProcessOrderLineItem_zeroIDs(t *testing.T) {
	var s stubStock
	item := &events.OrderItemEvent{ProductId: 0, Quantity: 1}
	if err := ProcessOrderLineItem(context.Background(), &s, item); !errors.Is(err, ErrOrderLineInvalid) {
		t.Fatalf("got %v", err)
	}
	item2 := &events.OrderItemEvent{ProductId: 1, Quantity: 0}
	if err := ProcessOrderLineItem(context.Background(), &s, item2); !errors.Is(err, ErrOrderLineInvalid) {
		t.Fatalf("got %v", err)
	}
	if s.getN != 0 || s.updateN != 0 {
		t.Fatalf("expected no calls")
	}
}

func TestProcessOrderLineItem_getFails_noUpdate(t *testing.T) {
	s := &stubStock{getErr: errors.New("db down")}
	pid := uint64(5)
	item := &events.OrderItemEvent{ProductId: pid, Quantity: 1}
	err := ProcessOrderLineItem(context.Background(), s, item)
	if err == nil || errors.Is(err, ErrOrderLineInvalid) {
		t.Fatalf("expected wrapped get error, got %v", err)
	}
	if s.getN != 1 || s.updateN != 0 {
		t.Fatalf("getN=%d updateN=%d want 1,0", s.getN, s.updateN)
	}
}

func TestProcessOrderLineItem_insufficientStock_noUpdate(t *testing.T) {
	s := &stubStock{product: domain.Product{ID: 1, Stock: 2}}
	item := &events.OrderItemEvent{ProductId: 1, Quantity: 5}
	err := ProcessOrderLineItem(context.Background(), s, item)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("got %v want ErrInsufficientStock", err)
	}
	if s.getN != 1 || s.updateN != 0 {
		t.Fatalf("getN=%d updateN=%d want 1,0", s.getN, s.updateN)
	}
}

func TestProcessOrderLineItem_success(t *testing.T) {
	s := &stubStock{product: domain.Product{ID: 9, Stock: 10}}
	item := &events.OrderItemEvent{ProductId: 9, Quantity: 3}
	if err := ProcessOrderLineItem(context.Background(), s, item); err != nil {
		t.Fatal(err)
	}
	if s.getN != 1 || s.updateN != 1 {
		t.Fatalf("getN=%d updateN=%d want 1,1", s.getN, s.updateN)
	}
	if s.lastStock != 7 {
		t.Fatalf("lastStock=%d want 7", s.lastStock)
	}
}

func TestProcessOrderLineItem_updateFails(t *testing.T) {
	s := &stubStock{
		product:   domain.Product{ID: 3, Stock: 10},
		updateErr: errors.New("constraint"),
	}
	item := &events.OrderItemEvent{ProductId: 3, Quantity: 1}
	err := ProcessOrderLineItem(context.Background(), s, item)
	if err == nil || errors.Is(err, ErrOrderLineInvalid) || errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected update error, got %v", err)
	}
	if s.updateN != 1 {
		t.Fatalf("update should have been attempted")
	}
}
