package gql

import (
	"context"
	"sync"
	"time"

	libpack_logger "github.com/lukaszraczylo/go-simple-graphql/logging"
)

// warmupConnectionPool pre-creates connections by executing warmup queries
// This establishes connections in the pool before actual traffic arrives
func (b *BaseClient) warmupConnectionPool() {
	b.Logger.Info(&libpack_logger.LogMessage{
		Message: "Warming up connection pool",
		Pairs: map[string]interface{}{
			"pool_size":    b.pool_size,
			"warmup_query": b.pool_warmup_query,
			"endpoint":     b.endpoint,
		},
	})

	startTime := time.Now()
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	// Pre-create connections concurrently
	for i := 0; i < b.pool_size; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Execute warmup query to establish connection
			_, err := b.Query(b.pool_warmup_query, nil, nil)

			mu.Lock()
			if err != nil {
				failCount++
				b.Logger.Warning(&libpack_logger.LogMessage{
					Message: "Failed to warm up connection",
					Pairs: map[string]interface{}{
						"connection_index": index,
						"error":            err.Error(),
					},
				})
			} else {
				successCount++
				b.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Connection warmed up successfully",
					Pairs: map[string]interface{}{
						"connection_index": index,
					},
				})
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	b.Logger.Info(&libpack_logger.LogMessage{
		Message: "Connection pool warmup completed",
		Pairs: map[string]interface{}{
			"successful":  successCount,
			"failed":      failCount,
			"total":       b.pool_size,
			"duration_ms": duration.Milliseconds(),
		},
	})
}

// startPoolHealthMonitor starts a background goroutine that periodically
// checks and maintains pool health by sending health check queries
func (b *BaseClient) startPoolHealthMonitor() {
	b.Logger.Info(&libpack_logger.LogMessage{
		Message: "Starting connection pool health monitor",
		Pairs: map[string]interface{}{
			"check_interval": b.pool_health_interval,
			"pool_size":      b.pool_size,
		},
	})

	go func() {
		ticker := time.NewTicker(b.pool_health_interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				b.performPoolHealthCheck()
			case <-b.pool_stop:
				b.Logger.Info(&libpack_logger.LogMessage{
					Message: "Stopping connection pool health monitor",
					Pairs:   nil,
				})
				return
			}
		}
	}()
}

// performPoolHealthCheck executes health check queries to maintain pool connections
// This keeps connections alive and detects/replaces dead connections
func (b *BaseClient) performPoolHealthCheck() {
	b.Logger.Debug(&libpack_logger.LogMessage{
		Message: "Performing pool health check",
		Pairs: map[string]interface{}{
			"pool_size": b.pool_size,
		},
	})

	startTime := time.Now()
	var wg sync.WaitGroup
	healthyCount := 0
	unhealthyCount := 0
	var mu sync.Mutex

	// Check a subset of connections to ensure pool health
	// We check pool_size/2 to balance between thoroughness and overhead
	checkCount := max(1, b.pool_size/2)

	for i := 0; i < checkCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Create a context with timeout for health check
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use a channel to handle timeout
			done := make(chan error, 1)
			go func() {
				_, err := b.Query(b.pool_warmup_query, nil, nil)
				done <- err
			}()

			select {
			case err := <-done:
				mu.Lock()
				if err != nil {
					unhealthyCount++
					b.Logger.Warning(&libpack_logger.LogMessage{
						Message: "Pool health check failed",
						Pairs: map[string]interface{}{
							"check_index": index,
							"error":       err.Error(),
						},
					})
				} else {
					healthyCount++
				}
				mu.Unlock()
			case <-ctx.Done():
				mu.Lock()
				unhealthyCount++
				b.Logger.Warning(&libpack_logger.LogMessage{
					Message: "Pool health check timeout",
					Pairs: map[string]interface{}{
						"check_index": index,
					},
				})
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	b.Logger.Debug(&libpack_logger.LogMessage{
		Message: "Pool health check completed",
		Pairs: map[string]interface{}{
			"healthy":     healthyCount,
			"unhealthy":   unhealthyCount,
			"checked":     checkCount,
			"duration_ms": duration.Milliseconds(),
		},
	})

	// If more than 50% of checks failed, trigger a pool refresh
	if unhealthyCount > checkCount/2 {
		b.Logger.Warning(&libpack_logger.LogMessage{
			Message: "Pool health degraded, triggering refresh",
			Pairs: map[string]interface{}{
				"unhealthy_ratio": float64(unhealthyCount) / float64(checkCount),
			},
		})
		b.refreshConnectionPool()
	}
}

// refreshConnectionPool attempts to refresh the connection pool
// by executing warmup queries to replace stale connections
func (b *BaseClient) refreshConnectionPool() {
	b.Logger.Info(&libpack_logger.LogMessage{
		Message: "Refreshing connection pool",
		Pairs: map[string]interface{}{
			"pool_size": b.pool_size,
		},
	})

	var wg sync.WaitGroup
	refreshCount := max(1, b.pool_size/3) // Refresh 1/3 of pool

	for i := 0; i < refreshCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			_, err := b.Query(b.pool_warmup_query, nil, nil)
			if err != nil {
				b.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Pool refresh connection failed",
					Pairs: map[string]interface{}{
						"connection_index": index,
						"error":            err.Error(),
					},
				})
			} else {
				b.Logger.Debug(&libpack_logger.LogMessage{
					Message: "Pool connection refreshed",
					Pairs: map[string]interface{}{
						"connection_index": index,
					},
				})
			}
		}(i)
	}

	wg.Wait()
}

// StopPoolMonitor gracefully stops the pool health monitor
// Call this when shutting down the application
func (b *BaseClient) StopPoolMonitor() {
	if b.pool_warmup_enabled {
		b.Logger.Info(&libpack_logger.LogMessage{
			Message: "Stopping pool monitor",
			Pairs:   nil,
		})
		select {
		case b.pool_stop <- true:
		default:
			// Channel already has a value or monitor already stopped
		}
	}
}
