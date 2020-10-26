package main

import (
	"testing"
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
