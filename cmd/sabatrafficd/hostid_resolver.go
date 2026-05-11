package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mackerelio-labs/sabatrafficd/internal/config"
)

type customIdentifierFinder interface {
	FindHostByCustomIdentifierContext(ctx context.Context, customIdentifier string) (string, error)
}

func resolveCollectorHostIDs(ctx context.Context, collectors []*config.CollectorConfig, finder customIdentifierFinder) []*config.CollectorConfig {
	resolved := make([]*config.CollectorConfig, 0, len(collectors))
	for i := range collectors {
		if collectors[i].HostID != "" {
			resolved = append(resolved, collectors[i])
			continue
		}

		hostID, err := findHostIDByCustomIdentifier(ctx, finder, collectors[i].CustomIdentifier)
		if err != nil {
			slog.WarnContext(ctx, "skip collector because failed resolve host-id",
				slog.String("host", collectors[i].SNMP.Host),
				slog.String("custom-identifier", collectors[i].CustomIdentifier),
				slog.String("error", err.Error()),
			)
			continue
		}
		collectors[i].HostID = hostID
		resolved = append(resolved, collectors[i])
	}

	return resolved
}

func findHostIDByCustomIdentifier(ctx context.Context, finder customIdentifierFinder, customIdentifier string) (string, error) {
	var err error
	for range 3 {
		var hostID string
		hostID, err = finder.FindHostByCustomIdentifierContext(ctx, customIdentifier)
		if err == nil {
			return hostID, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return "", fmt.Errorf("host id is invalid, custom-identifier: %s, error: %w", customIdentifier, err)
}
