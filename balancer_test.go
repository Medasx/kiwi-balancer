package main

import (
	"context"
	"testing"
	"time"
)

var divideResourcesTestCases = []struct {
	name  string
	input struct {
		chunks int
		jobs   []*job
	}
	result map[int]int
}{
	{
		name: "Basic",
		input: struct {
			chunks int
			jobs   []*job
		}{
			chunks: 100,
			jobs: []*job{
				{priority: 1},
				{priority: 1},
				{priority: 2},
			},
		},
		result: map[int]int{
			0: 25,
			1: 25,
			2: 50,
		},
	},
	{
		name: "EqualPriorities",
		input: struct {
			chunks int
			jobs   []*job
		}{
			chunks: 100,
			jobs: []*job{
				{priority: 1},
				{priority: 1},
				{priority: 1},
			},
		},
		result: map[int]int{
			0: 34,
			1: 33,
			2: 33,
		},
	},
	{
		name: "NotEnoughChunks",
		input: struct {
			chunks int
			jobs   []*job
		}{
			chunks: 10,
			jobs: []*job{
				{priority: 1},
				{priority: 2},
				{priority: 3},
				{priority: 3},
				{priority: 3},
			},
		},
		result: map[int]int{
			1: 1,
			2: 3,
			3: 3,
			4: 3,
		},
	},
}

func TestDivideResources(t *testing.T) {
	b := NewBalancer(&TheExpensiveFragileService{}, 23)
	for _, item := range divideResourcesTestCases {
		t.Run(item.name, func(t *testing.T) {
			res := b.divideResources(item.input.jobs, item.input.chunks)
			for k1, v1 := range item.result {
				if v2, ok := res[k1]; ok {
					if v1 != v2 {
						t.Errorf("index %d expected %d got %d", k1, v1, v2)
					}
				} else {
					t.Errorf("index %d not found", k1)
				}
			}
		})

	}
}

var normalizePrioritiesTestCases = []struct {
	name   string
	input  map[int][]int
	result map[int][]int
}{
	{
		name: "Basic",
		input: map[int][]int{
			7:  {0, 2},
			5:  {1, 3},
			10: {4},
		},
		result: map[int][]int{
			2: {0, 2},
			1: {1, 3},
			3: {4},
		},
	},
}

func TestNormalizePriorities(t *testing.T) {
	for _, item := range normalizePrioritiesTestCases {
		t.Run(item.name, func(t *testing.T) {
			res := normalizePriorities(item.input)
			for k1, v1 := range item.result {
				if v2, ok := res[k1]; ok {
					if len(v2) != len(v1) {
						t.Errorf("expected len %d got %d", len(v1), len(v2))
					}
					for i, item := range v1 {
						if item != v2[i] {
							t.Errorf("index %d expected %d got %d", i, item, v2[i])
						}
					}

				} else {
					t.Errorf("index %d not found", k1)
				}
			}
		})

	}
}

func TestBalancer(t *testing.T) {
	b := NewBalancer(&TheExpensiveFragileService{}, 0)
	c := []*customer{
		{weight: 5, workload: 50, needsService: time.NewTicker(time.Second), done: make(chan struct{}, 1)},
		{weight: 5, workload: 50, needsService: time.NewTicker(time.Second), done: make(chan struct{}, 1)},
		{weight: 10, workload: 50, needsService: time.NewTicker(time.Second), done: make(chan struct{}, 1)},
	}
	for _, customer := range c {
		b.Register(context.Background(), customer)
	}
	b.availableChunks <- 100

	select {
	case <-c[0].Done():
		t.Errorf("customer %d should not be done first", 0)
	case <-c[1].Done():
		t.Errorf("customer %d should not be done first", 1)
	case <-c[2].Done():
		t.Log("highest weight ended first")

	}

}
