package main

import (
	"fmt"
	"sync"
	"time"
	"github.com/gofiber/fiber/v2"
)

type RequestRateTracker struct {
	mu           sync.RWMutex
	requests     int
	lastSnapshot time.Time
	rate         int
	stopCh       chan struct{}
}

func NewRequestRateTracker() *RequestRateTracker {
	tracker := &RequestRateTracker{
		lastSnapshot: time.Now(),
		stopCh:       make(chan struct{}),
	}
	go tracker.updateLoop()
	return tracker
}

func (rt *RequestRateTracker) updateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rt.CalculateRequestRate()
		case <-rt.stopCh:
			return
		}
	}
}

func (rt *RequestRateTracker) CalculateRequestRate() {
	now := time.Now()
	elapsed := now.Sub(rt.lastSnapshot).Seconds()

	rt.mu.Lock()
	rt.rate = int(float64(rt.requests) / elapsed)
	rt.lastSnapshot = now
	rt.requests = 0
	rt.mu.Unlock()
}

func (rt *RequestRateTracker) Middleware(c *fiber.Ctx) error {
	rt.mu.Lock()
	rt.requests++
	rt.mu.Unlock()

	return c.Next()
}

func (rt *RequestRateTracker) GetRequestRate() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.rate
}

func (rt *RequestRateTracker) Stop() {
	close(rt.stopCh)
}

func main() {
	app := fiber.New()

	tracker := NewRequestRateTracker()
	defer tracker.Stop()

	app.Use(tracker.Middleware)

	app.Get("/", func(c *fiber.Ctx) error {
		currentRate := tracker.GetRequestRate()
		return c.JSON(map[string]interface{}{
			"requests_per_second": currentRate,
		})
	})

	if err := app.Listen(":3000"); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
