package main

import (
	"context"
	"math/rand"
	"time"
)

// TheExpensiveFragileService is a service that we need to utilize as much as we can, since it's expensive to run,
// but on the other hand it's very fragile so we can't just flood it with thousands of requests per second.
type TheExpensiveFragileService struct{}

// Process a single work chunk and return error if occurred.
func (t *TheExpensiveFragileService) Process(_ context.Context, _ int) error {
	// do not implement me, just imagine there's huge, complex, almost extra-terrestrial logic here
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	return nil
}
