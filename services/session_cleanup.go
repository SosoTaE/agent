package services

import (
	"context"
	"log/slog"
	"time"
)

// StartSessionCleanup starts a background goroutine that periodically cleans up expired sessions
func StartSessionCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Run cleanup every hour
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Session cleanup stopped")
				return
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				count, err := CleanupExpiredSessions(cleanupCtx)
				if err != nil {
					slog.Error("Failed to cleanup expired sessions", "error", err)
				} else if count > 0 {
					slog.Info("Cleaned up expired sessions", "count", count)
				}
				cancel()
			}
		}
	}()

	slog.Info("Session cleanup started")
}
