package main

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/maintainers/experiments/keptain/pkg/store"
	"sigs.k8s.io/maintainers/experiments/keptain/pkg/website"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Initialize the KEP repository
	kepRepo, err := store.NewRepository(ctx, "enhancements")
	if err != nil {
		return fmt.Errorf("error creating KEP repository: %w", err)
	}

	// Start the web server
	server := website.NewServer(kepRepo)
	return server.Run(":8080")
}
