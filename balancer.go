package main

import (
	"context"
	"fmt"
)

// Balancer should utilize the expensive service as much as it can while making sure it does not overflow it with
// work chunks. There is a hard limit for max parallel chunks that can't ever be crossed since there's a SLO defined
// and we don't want to make the expensive service people angry.
//
// There can be arbitrary number of Customers registered at any given time, each with it's own weight. Make sure to
// correctly assign processing capacity to a customer based on other customers currently in process.
//
// To give an example of this, imagine there's a maximum number of work chunks set to 100 and there are two customers
// registered, both with the same priority. When they are both served in parallel, each of them gets to send
// 50 chunks at the same time.
//
// In the same scenario, if there were two customers with priority 1 and one customer with priority 2, the first
// two would be allowed to send 25 chunks and the other one would send 50. It's likely that the one sending 50 would
// be served faster, finishing the work early, meaning that it would no longer be necessary that those first two
// customers only send 25 each but can and should use the remaining capacity and send 50 again.
type Balancer struct {
	availableChunks chan int
	jobs            chan *job // could be easily replaced with interface

	service *TheExpensiveFragileService
}

// NewBalancer takes a reference to the main service and a maximum number of work chunks that can be fed through it
// in parallel.
func NewBalancer(service *TheExpensiveFragileService, maxChunks int) *Balancer {
	availableChunks := make(chan int, 1)
	availableChunks <- maxChunks

	b := &Balancer{jobs: make(chan *job, 100), availableChunks: availableChunks, service: service}
	b.start()

	return b
}

// Register a customer to the balancer and start processing it's work chunks through the service.
func (b *Balancer) Register(ctx context.Context, customer Customer) {
	b.jobs <- NewJob(ctx, customer, func() {
		if err := b.service.Process(ctx, 0); err != nil {
			fmt.Printf("processing error: %v", err)
		}
	})

}

// starts non blocking infinite loop that handles customers
func (b *Balancer) start() {
	go func() {
		for chunks, queue := 0, []*job{}; ; {
			// block processing until there are available chunks and jobs to process
			select {
			case x := <-b.availableChunks:
				chunks += x
			case x := <-b.jobs:
				queue = append(queue, x)
			}

			// read all ready to send items from channels
			moreChunks, jobs := flushChannels(b.availableChunks, b.jobs)
			chunks += moreChunks
			queue = append(queue, jobs...)

			if chunks == 0 || len(queue) == 0 {
				// prevent from processing with no chunks available or no jobs to process
				continue
			}
			indexes := b.divideResources(queue, chunks)
			for index, assignResources := range indexes {
				b.free(queue[index].processResources(assignResources))
			}
			if len(indexes) == 0 {
				// happens when all job in queue are completed
				continue
			}

			var incomplete []*job
			for _, j := range queue {
				if !j.IsComplete() {
					incomplete = append(incomplete, j)
					// jobs completed in between cycles will be ignored
				}
			}

			// reset resources
			queue = incomplete
			chunks = 0
		}
	}()
}

// divideResources correctly divide chunks among jobs
func (b *Balancer) divideResources(jobs []*job, freeChunks int) map[int]int {
	priorities := make(map[int][]int)
	for index, item := range jobs {
		if item.IsComplete() {
			continue
		}
		priority := item.priority

		if _, ok := priorities[priority]; !ok {
			priorities[priority] = make([]int, 0)
		}
		priorities[priority] = append(priorities[priority], index)
	}
	normalized := normalizePriorities(priorities)
	//fmt.Println("p ", priorities, jobs[0])
	return b.assignResources(normalized, freeChunks)
}

// assignResources returns map of jobIndexes and chunks assigned to them
func (b *Balancer) assignResources(in map[int][]int, chunks int) (result map[int]int) {
	result = make(map[int]int)
	b.assign(in, result, chunks, 0)
	return
}

// recursive assigning chunks to jobs, with increasing level smallest priorities are ignored
func (b *Balancer) assign(in map[int][]int, result map[int]int, chunks int, level int) {
	var priorities []int
	for priority, _ := range in {
		// ignore smallest priorities with increasing level
		// priority 1 is ignored if level >= 1
		if priority-level > 0 {
			priorities = append(priorities, priority)
		}
	}
	if len(priorities) == 0 {
		return
	}

	if len(priorities) == 1 {
		if chunks < len(in[priorities[0]]) {
			// number of chunks is less than number of jobs with top (or only) priority
			for _, x := range in[priorities[0]][:chunks] {
				if _, ok := result[x]; !ok {
					result[x] = 0
				}
				result[x] += 1
			}
			return
		}
	}

	// sum number of jobs multiplied by priority per every priority in slice
	sum := 0
	for _, priority := range priorities {
		sum += (priority - level) * len(in[priority])
	}

	// there isn't enough chunks to assign at least 1 to each job with lowest priority
	if sum > chunks {
		// recursive call with increased level will ignore lowest priority
		b.assign(in, result, chunks, level+1)
		return
	}

	// calculate coefficient that will be multiplied by priority
	// leftOvers will be assign later
	leftOvers := chunks % sum
	coefficient := (chunks - leftOvers) / sum

	// assign resources
	for _, priority := range priorities {
		resources := (priority - level) * coefficient
		for _, x := range in[priority] {
			if _, ok := result[x]; !ok {
				result[x] = 0
			}
			result[x] += resources
		}
	}

	// assign leftover if there are any
	if leftOvers != 0 {
		b.assign(in, result, leftOvers, level)

	}
	return
}

// free read freed chunks without blocking and make them available
func (b *Balancer) free(in chan int) {
	go func() {
		c := <-in
		b.availableChunks <- c
	}()
}

// flushChannels reads all items from channels without blocking
func flushChannels(availableChunks chan int, jobs chan *job) (chunks int, queue []*job) {
loop:
	for { // read all chunks from channel
		select {
		case x := <-availableChunks:
			chunks += x
		case x := <-jobs:
			queue = append(queue, x)
		default:
			break loop
		}
	}
	return
}
