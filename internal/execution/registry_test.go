package execution_test

import (
	"context"
	"errors"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/execution"
)

type stubBroker struct{}

func (stubBroker) SubmitOrder(context.Context, *domain.Order) (string, error) {
	return "", nil
}

func (stubBroker) CancelOrder(context.Context, string) error {
	return nil
}

func (stubBroker) GetOrderStatus(context.Context, string) (domain.OrderStatus, error) {
	return domain.OrderStatusSubmitted, nil
}

func (stubBroker) GetPositions(context.Context) ([]domain.Position, error) {
	return nil, nil
}

func (stubBroker) GetAccountBalance(context.Context) (execution.Balance, error) {
	return execution.Balance{}, nil
}

func TestRegistryRegisterAndResolve(t *testing.T) {
	registry := execution.NewRegistry()
	stockBroker := stubBroker{}
	cryptoBroker := stubBroker{}

	if err := registry.Register(" Stock ", stockBroker); err != nil {
		t.Fatalf("Register(stock) error = %v", err)
	}
	if err := registry.Register(domain.MarketTypeCrypto, cryptoBroker); err != nil {
		t.Fatalf("Register(crypto) error = %v", err)
	}

	gotStock, ok := registry.Get(domain.MarketTypeStock)
	if !ok {
		t.Fatal("Get(stock) ok = false, want true")
	}
	if _, ok := gotStock.(stubBroker); !ok {
		t.Fatalf("Get(stock) broker type = %T, want stubBroker", gotStock)
	}

	gotCrypto, err := registry.Resolve("CRYPTO")
	if err != nil {
		t.Fatalf("Resolve(crypto) error = %v", err)
	}
	if _, ok := gotCrypto.(stubBroker); !ok {
		t.Fatalf("Resolve(crypto) broker type = %T, want stubBroker", gotCrypto)
	}
}

func TestRegistryResolveErrors(t *testing.T) {
	registry := execution.NewRegistry()

	_, err := registry.Resolve("missing")
	if !errors.Is(err, execution.ErrBrokerNotFound) {
		t.Fatalf("Resolve() error = %v, want ErrBrokerNotFound", err)
	}
}

func TestRegistryRegisterRejectsBlankMarketTypeOrMissingBroker(t *testing.T) {
	tests := []struct {
		name       string
		marketType domain.MarketType
		broker     execution.Broker
		want       string
	}{
		{
			name:       "blank market type",
			marketType: " ",
			broker:     stubBroker{},
			want:       "execution market type is required",
		},
		{
			name:       "missing broker",
			marketType: domain.MarketTypePolymarket,
			broker:     nil,
			want:       "execution broker is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := execution.NewRegistry()

			err := registry.Register(tc.marketType, tc.broker)
			if err == nil {
				t.Fatal("Register() error = nil, want non-nil")
			}
			if err.Error() != tc.want {
				t.Fatalf("Register() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}
