package main

import (
	"context"
	"math/rand"
	"time"
)

// Customer defines methods that represent actual customer that needs to be served by someone. They can request to
// be served, provide workload and state how important they are (meaning how fast they need to be served).
type Customer interface {
	// Notify that the customer needs to be served. Returns channel delivering periodic ticks.
	Notify() <-chan time.Time
	// Workload returns a channel of work chunks that are to be processed through TheExpensiveFragileService.
	Workload(ctx context.Context) chan int
	// Weight is unit-less number that determines how much processing capacity should a customer be allocated
	// when running in parallel with other customers. The higher the weight, the more capacity the customer gets.
	Weight() int
	// Stop the customer and clean up.
	Stop()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	nbCustomers := rand.Intn(10)+1
	customers := make([]Customer, 0, nbCustomers)
	for i := 0; i < nbCustomers; i++ {
		customers = append(customers, NewCustomer())
	}

	defer func() {
		for i := range customers {
			customers[i].Stop()
		}
	}()

	tefs := &TheExpensiveFragileService{}

	maxParallel := rand.Intn(150)
	b := NewBalancer(tefs, maxParallel)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := range customers {
		go func(i int) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-customers[i].Notify():
					b.Register(ctx, customers[i])
				}
			}
		}(i)
	}
	<-ctx.Done()
}
