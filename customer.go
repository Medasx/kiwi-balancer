package main

import (
	"context"
	"math/rand"
	"time"
)

// customer implements Customer interface.
type customer struct {
	needsService *time.Ticker
	workload     int
	weight       int
	done         chan struct{} // added for better testing
}

// NewCustomer initializes the customer.
func NewCustomer() Customer {
	return &customer{
		needsService: time.NewTicker(time.Duration(1+rand.Intn(3)) * time.Second),
		workload:     rand.Intn(100),
		weight:       rand.Intn(10),
		done:         make(chan struct{}, 1),
	}
}

// Stop the underlying needsService.
func (c *customer) Stop() {
	c.needsService.Stop()
}

// Notify that someone should read customer's workload and start processing.
// Returns a channel through which periodic ticks are delivered.
func (c *customer) Notify() <-chan time.Time {
	return c.needsService.C
}

// Workload feeds work chunks through returned channel until there is no more work to be fed or ctx was Done.
func (c *customer) Workload(ctx context.Context) chan int {
	workload := make(chan int)
	go func() {
		defer close(c.done)
		defer close(workload)

		load := c.workload
		for {
			select {
			case <-ctx.Done():
				return
			case workload <- load:
				if load -= 1; load == 0 {
					c.done <- struct{}{}

					return
				}
			}
		}
	}()
	return workload
}

// Weight of this customer, i.e. how important they are.
func (c *customer) Weight() int {
	return c.weight
}

// Done returns a channel that's closed when work done. Added
func (c *customer) Done() <-chan struct{} {
	return c.done
}
