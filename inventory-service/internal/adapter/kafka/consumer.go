package kafka

import (
	"errors"
	"log"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/19parwiz/inventory-service/internal/usecase"
	events "github.com/19parwiz/inventory-service/protos/gen/golang"
)

type Consumer struct {
	usecase ProductStock
	Topic   string
}

func NewConsumer(usecase *usecase.Product, topic string) *Consumer {
	return &Consumer{usecase: usecase, Topic: topic}
}

func (h *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		var event events.OrderCreatedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			log.Printf("kafka consumer: unmarshal OrderCreatedEvent: %v", err)
			continue
		}

		for _, item := range event.GetItems() {
			err := ProcessOrderLineItem(session.Context(), h.usecase, item)
			if err == nil {
				continue
			}
			switch {
			case errors.Is(err, ErrOrderLineInvalid):
				if item == nil {
					log.Printf("kafka consumer: skip nil line item")
				} else {
					log.Printf("kafka consumer: skip line item (product_id=%d quantity=%d)", item.GetProductId(), item.GetQuantity())
				}
			case errors.Is(err, ErrInsufficientStock):
				log.Printf("kafka consumer: %v", err)
			default:
				log.Printf("kafka consumer: %v", err)
			}
		}

		session.MarkMessage(message, "")
	}
	return nil
}
