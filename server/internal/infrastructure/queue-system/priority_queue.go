package queuesystem

import (
	"sort"
	"sync"
)

// Task represents an item in the priority queue.
type Task struct {
	ID       string
	Priority int
	Index    int // Used to maintain FIFO order for tasks with the same priority
}

// PriorityQueue implements a threshold-based fair priority queue.
type PriorityQueue struct {
	mu              sync.Mutex
	tasks           map[int][]*Task // Stores tasks by priority
	thresholds      map[int]int     // Defines how many tasks of a given priority must be processed
	processedCounts map[int]int     // Tracks processed tasks for each priority against thresholds
	priorities      []int           // Sorted list of unique priority levels (descending)
	currentIndex    int             // Current priority level index in the priorities slice
}

// NewPriorityQueue creates a new PriorityQueue.
// thresholds: map[priority]count. count = 0 means drain all tasks of this priority.
func NewPriorityQueue(thresholds map[int]int) *PriorityQueue {
	pq := &PriorityQueue{
		tasks:           make(map[int][]*Task),
		thresholds:      thresholds,
		processedCounts: make(map[int]int),
		priorities:      make([]int, 0),
		currentIndex:    0,
	}
	return pq
}

// Enqueue adds a task to the queue.
func (pq *PriorityQueue) Enqueue(task Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if _, ok := pq.tasks[task.Priority]; !ok {
		pq.tasks[task.Priority] = make([]*Task, 0)
		// Add new priority and keep priorities sorted
		pq.addPriority(task.Priority)
	}
	pq.tasks[task.Priority] = append(pq.tasks[task.Priority], &task)
}

func (pq *PriorityQueue) addPriority(priority int) {
	// This function is called under lock
	for _, p := range pq.priorities {
		if p == priority {
			return // Priority already exists
		}
	}
	pq.priorities = append(pq.priorities, priority)
	// Sort priorities in descending order (higher value = higher priority)
	sort.Sort(sort.Reverse(sort.IntSlice(pq.priorities)))
	// Reset currentIndex if a higher priority was added
	pq.currentIndex = 0
}

// Dequeue removes and returns the next task according to fairness rules.
// Returns nil if the queue is empty.
func (pq *PriorityQueue) Dequeue() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.priorities) == 0 {
		return nil // No priorities, queue is empty or was never populated
	}

	initialIndex := pq.currentIndex
	processedThisLoop := false

	for i := 0; i < len(pq.priorities)*2; i++ { // Loop at most twice through all priorities
		if len(pq.priorities) == 0 {
			return nil
		}

		currentPriority := pq.priorities[pq.currentIndex]

		if queue, ok := pq.tasks[currentPriority]; ok && len(queue) > 0 {
			threshold := pq.thresholds[currentPriority] // If currentPriority not in map, threshold is 0 (drain)
			processed := pq.processedCounts[currentPriority]

			if threshold == 0 || processed < threshold {
				task := queue[0]
				pq.tasks[currentPriority] = queue[1:]

				pq.processedCounts[currentPriority] = processed + 1
				processedThisLoop = true

				if threshold > 0 && pq.processedCounts[currentPriority] >= threshold {
					pq.processedCounts[currentPriority] = 0
					pq.currentIndex = (pq.currentIndex + 1) % len(pq.priorities)
				}
				return task
			}
		}

		pq.currentIndex = (pq.currentIndex + 1) % len(pq.priorities)

		if pq.currentIndex == initialIndex && !processedThisLoop {
			if !pq.hasTasks() {
				return nil
			}

			if i >= len(pq.priorities)-1 {
				allQueuesThresholdMetOrEmpty := true
				// Use different variable names for p, tasks, ok in this inner loop
				// to avoid any possible (though unlikely for syntax error) confusion.
				for _, p_check := range pq.priorities {
					if tasks_check, ok_check := pq.tasks[p_check]; ok_check && len(tasks_check) > 0 {
						// Check threshold for p_check; if not in map, it's 0.
						if pq.thresholds[p_check] == 0 || pq.processedCounts[p_check] < pq.thresholds[p_check] {
							allQueuesThresholdMetOrEmpty = false
							break
						}
					}
				}
				if allQueuesThresholdMetOrEmpty {
					for priorityKey := range pq.processedCounts {
						if pq.thresholds[priorityKey] > 0 {
							pq.processedCounts[priorityKey] = 0
						}
					}
				}
			}
		}
	}
	return nil
}

// Helper to check if any tasks exist in any queue
// This function is called by Dequeue, which already holds the lock.
func (pq *PriorityQueue) hasTasks() bool {
	for _, queue := range pq.tasks {
		if len(queue) > 0 {
			return true
		}
	}
	return false
}

// Len returns the total number of tasks in the queue.
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	count := 0
	for _, queue := range pq.tasks {
		count += len(queue)
	}
	return count
}
