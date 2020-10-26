package main

import (
	"context"
	"sort"
	"sync"
)

func NewJob(ctx context.Context, c Customer, process func()) *job {
	return &job{
		Customer: c,
		workload: c.Workload(ctx),
		priority: c.Weight() + 1, // increased by one avoid zero value
		process:  process,
	}
}

// job wrap Customer interface, balancer internally uses job instead of customer
type job struct {
	Customer

	sync.Once
	complete bool

	// process a single work chunk
	// func is defined during registration, job doesn't need reference to service
	process func()
	// customer weight increased by 1 to avoid 0 value
	priority int
	// store customer workload since Customer.Workload() always returns new channel
	workload chan int
}

// Complete sets complete flag to true
func (j *job) Complete() {
	j.Do(
		func() {
			j.Customer.Stop()
			j.complete = true
		})
}

// IsComplete return complete flag
func (j *job) IsComplete() bool {
	return j.complete
}

// processResources takes parallel resources assigned to job and process, returns channel of freed resources
// all assigned resources are freed after all parallel process are done
func (j *job) processResources(resources int) chan int {
	done := make(chan int, resources)
	go func() {
		defer close(done)
		wg := &sync.WaitGroup{}
		for i := 0; i < resources; i++ {
			wg.Add(1)
			go func() {
				if !j.IsComplete() {
					if _, ok := <-j.workload; ok {
						j.process()
					} else {
						j.Complete()
					}
				}
				wg.Done()

			}()

		}

		wg.Wait()
		done <- resources
	}()
	return done
}



// normalizePriorities takes map of `priorities: jobIndexes` and turns priorities into sequence of with {1, 2, ..., m}
func normalizePriorities(in map[int][]int) map[int][]int {
	var priorities []int
	for priority, _ := range in {
		priorities = append(priorities, priority)
	}

	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i] < priorities[j]
	})
	normalizedMap := make(map[int][]int)
	for i := 0; i < len(priorities); i++ {
		normalizedMap[i+1] = in[priorities[i]]
	}
	return normalizedMap
}

