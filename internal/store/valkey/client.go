package valkey

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"

	"github.com/maraichr/lattice/internal/config"
)

func NewClient(cfg config.ValkeyConfig) (valkey.Client, error) {
	opts := valkey.ClientOption{
		InitAddress: []string{cfg.Addr},
	}
	if cfg.Password != "" {
		opts.Password = cfg.Password
	}

	client, err := valkey.NewClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create valkey client: %w", err)
	}

	// Verify connectivity
	ctx := context.Background()
	resp := client.Do(ctx, client.B().Ping().Build())
	if err := resp.Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping valkey: %w", err)
	}

	return client, nil
}
