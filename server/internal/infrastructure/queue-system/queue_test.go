package queuesystem

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestPriorityQueue_BasicEnqueueDequeue(t *testing.T) {
	thresholds := map[int]int{1: 0, 0: 0} // Drain all
	pq := NewPriorityQueue(thresholds)

	if pq.Len() != 0 {
		t.Errorf("Expected new queue length to be 0, got %d", pq.Len())
	}

	task1 := Task{ID: "1", Priority: 1, Index: 0}
	task2 := Task{ID: "2", Priority: 0, Index: 1}
	task3 := Task{ID: "3", Priority: 1, Index: 2}

	pq.Enqueue(task1)
	if pq.Len() != 1 {
		t.Errorf("Expected queue length to be 1, got %d", pq.Len())
	}
	pq.Enqueue(task2)
	if pq.Len() != 2 {
		t.Errorf("Expected queue length to be 2, got %d", pq.Len())
	}
	pq.Enqueue(task3)
	if pq.Len() != 3 {
		t.Errorf("Expected queue length to be 3, got %d", pq.Len())
	}

	// Expected order: task1, task3 (Priority 1), then task2 (Priority 0)
	expectedOrder := []*Task{&task1, &task3, &task2}
	actualOrder := []*Task{}

	for i := 0; i < len(expectedOrder); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly at iteration %d", i)
		}
		actualOrder = append(actualOrder, task)
	}

	if !reflect.DeepEqual(actualOrder, expectedOrder) {
		t.Errorf("Expected dequeue order %v, got %v", formatTasks(expectedOrder), formatTasks(actualOrder))
	}

	if pq.Len() != 0 {
		t.Errorf("Expected queue length to be 0 after dequeuing all, got %d", pq.Len())
	}

	if task := pq.Dequeue(); task != nil {
		t.Errorf("Expected Dequeue on empty queue to return nil, got %v", task)
	}
}

func TestPriorityQueue_ThresholdLogic_Example(t *testing.T) {
	thresholds := map[int]int{
		2: 0, // must fully consume all priority 2 tasks before going to 1
		1: 4, // after 4 priority 1 tasks, allow one of priority 0
		0: 1, // after 1 of priority 0, allow lower (if it existed) / or loop back to higher
	}
	pq := NewPriorityQueue(thresholds)

	tasks := []Task{
		{ID: "A", Priority: 2, Index: 0}, {ID: "B", Priority: 2, Index: 1},
		{ID: "C", Priority: 1, Index: 2}, {ID: "D", Priority: 1, Index: 3},
		{ID: "E", Priority: 1, Index: 4}, {ID: "F", Priority: 1, Index: 5},
		{ID: "G", Priority: 1, Index: 6}, // 5th task of priority 1
		{ID: "H", Priority: 0, Index: 7},
		{ID: "I", Priority: 0, Index: 8},
		{ID: "J", Priority: 0, Index: 9},
	}

	for _, task := range tasks {
		pq.Enqueue(task)
	}

	expectedIDs := []string{"A", "B", "C", "D", "E", "F", "H", "G", "I", "J"}
	actualIDs := []string{}

	for i := 0; i < len(expectedIDs); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly. Expected %s, iteration %d. Got %d tasks so far: %v", expectedIDs[i], i, len(actualIDs), actualIDs)
		}
		actualIDs = append(actualIDs, task.ID)
	}

	if !reflect.DeepEqual(actualIDs, expectedIDs) {
		t.Errorf("Expected dequeue order %v, got %v", expectedIDs, actualIDs)
	}

	if pq.Len() != 0 {
		t.Errorf("Expected queue length to be 0 after dequeuing all, got %d", pq.Len())
	}
}


func TestPriorityQueue_OnlyOnePriority(t *testing.T) {
	thresholds := map[int]int{1: 2} // Threshold doesn't really matter if only one priority
	pq := NewPriorityQueue(thresholds)

	task1 := Task{ID: "A", Priority: 1, Index: 0}
	task2 := Task{ID: "B", Priority: 1, Index: 1}
	task3 := Task{ID: "C", Priority: 1, Index: 2}

	pq.Enqueue(task1)
	pq.Enqueue(task2)
	pq.Enqueue(task3)

	expectedOrder := []*Task{&task1, &task2, &task3}
	actualOrder := []*Task{}

	for i := 0; i < len(expectedOrder); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly at iteration %d", i)
		}
		actualOrder = append(actualOrder, task)
	}

	if !reflect.DeepEqual(actualOrder, expectedOrder) {
		t.Errorf("Expected dequeue order %v, got %v", formatTasks(expectedOrder), formatTasks(actualOrder))
	}
}

func TestPriorityQueue_EmptyQueue(t *testing.T) {
	pq := NewPriorityQueue(map[int]int{1: 1})
	if task := pq.Dequeue(); task != nil {
		t.Errorf("Expected Dequeue on empty queue to return nil, got %v", task)
	}
}

func TestPriorityQueue_ThresholdZeroForAll(t *testing.T) {
	pq := NewPriorityQueue(map[int]int{2: 0, 1: 0, 0: 0})
	tasks := []Task{
		{ID: "A", Priority: 0, Index: 0}, {ID: "B", Priority: 1, Index: 1},
		{ID: "C", Priority: 2, Index: 2}, {ID: "D", Priority: 1, Index: 3},
		{ID: "E", Priority: 0, Index: 4}, {ID: "F", Priority: 2, Index: 5},
	}
	for _, task := range tasks {
		pq.Enqueue(task)
	}

	// Expected: C, F (P2 drained), then B, D (P1 drained), then A, E (P0 drained)
	expectedIDs := []string{"C", "F", "B", "D", "A", "E"}
	actualIDs := []string{}

	for i := 0; i < len(expectedIDs); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly at iteration %d. Expected %s", i, expectedIDs[i])
		}
		actualIDs = append(actualIDs, task.ID)
	}

	if !reflect.DeepEqual(actualIDs, expectedIDs) {
		t.Errorf("Expected dequeue order %v, got %v", expectedIDs, actualIDs)
	}
}

func TestPriorityQueue_InterleavedEnqueueDequeue(t *testing.T) {
	pq := NewPriorityQueue(map[int]int{1: 1, 0: 1}) // Process 1 of P1, then 1 of P0

	pq.Enqueue(Task{ID: "A", Priority: 1, Index: 0})
	pq.Enqueue(Task{ID: "B", Priority: 1, Index: 1})
	pq.Enqueue(Task{ID: "C", Priority: 0, Index: 2})
	pq.Enqueue(Task{ID: "D", Priority: 0, Index: 3})

	// Expected: A (P1, count_1=1, next P0)
	// C (P0, count_0=1, next P1)
	// B (P1, count_1=1, next P0)
	// D (P0, count_0=1, next P1)
	expectedIDs := []string{"A", "C", "B", "D"}
	actualIDs := []string{}

	var taskLastDequeued *Task

	for i := 0; i < len(expectedIDs); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly at iteration %d. Expected %s", i, expectedIDs[i])
		}
		actualIDs = append(actualIDs, task.ID)
		taskLastDequeued = task
		// Simulate adding more tasks dynamically
		if task.ID == "A" {
			pq.Enqueue(Task{ID: "E", Priority: 1, Index: 4}) // Will be after B
		}
		if task.ID == "C" {
			pq.Enqueue(Task{ID: "F", Priority: 0, Index: 5}) // Will be after D
		}
	}

	if !reflect.DeepEqual(actualIDs, expectedIDs) {
		t.Errorf("Expected dequeue order %v, got %v for initial set. Last dequeued: %s", expectedIDs, actualIDs, formatTask(taskLastDequeued))
	}

	// Now dequeue E and F
	// After D, current index points to P1 (because D was P0, threshold met, count reset, index incremented).
	// P1 queue has E.
	// Expected: E (P1, count_1=1, next P0)
	// P0 queue has F.
	// F (P0, count_0=1, next P1)
	expectedIDsAfter := []string{"E", "F"}
	actualIDsAfter := []string{}
	for i := 0; i < len(expectedIDsAfter); i++ {
		task := pq.Dequeue()
		if task == nil {
			t.Fatalf("Dequeue returned nil unexpectedly for E/F part at iteration %d. Expected %s. PQ state: %d items. Priorities: %v. CurrentIndex: %d. Counts: %v", i, expectedIDsAfter[i], pq.Len(), pq.priorities, pq.currentIndex, pq.processedCounts)
		}
		actualIDsAfter = append(actualIDsAfter, task.ID)
	}
	if !reflect.DeepEqual(actualIDsAfter, expectedIDsAfter) {
		t.Errorf("Expected dequeue order %v, got %v for dynamically added tasks", expectedIDsAfter, actualIDsAfter)
	}

}

func TestPriorityQueue_NewHighestPriorityAdded(t *testing.T) {
	pq := NewPriorityQueue(map[int]int{2:1, 1:1, 0:1}) // Explicitly define threshold for P2
	pq.Enqueue(Task{ID: "A", Priority: 0, Index: 0})
	pq.Enqueue(Task{ID: "B", Priority: 1, Index: 1}) // B is P1

	task := pq.Dequeue() // Should be B (P1)
	if task == nil || task.ID != "B" {
		t.Fatalf("Expected B, got %v", formatTask(task))
	}
	// pq.processedCounts[1] = 1. pq.currentIndex should now point to P0 (index 1 of sorted [1,0])

	pq.Enqueue(Task{ID: "C", Priority: 2, Index: 2}) // New highest
	// pq.priorities becomes [2,1,0]. pq.currentIndex resets to 0 (points to P2)

	task = pq.Dequeue() // Should be C (P2)
	if task == nil || task.ID != "C" {
		t.Fatalf("Expected C, got %v. PQ state: curIdx %d, priorities %v, counts %v", formatTask(task), pq.currentIndex, pq.priorities, pq.processedCounts)
	}
	// pq.processedCounts[2] = 1. pq.currentIndex should now point to P1 (index 1 of [2,1,0])

	// P1 queue is empty. Dequeue should skip P1.
	// pq.processedCounts[1] should be reset (or remain 0 if it was already) because we are moving past it.
	// Then it moves to P0.
	task = pq.Dequeue() // Should be A (P0)
	if task == nil || task.ID != "A" {
		t.Fatalf("Expected A, got %v. PQ state: curIdx %d, priorities %v, counts %v", formatTask(task), pq.currentIndex, pq.priorities, pq.processedCounts)
	}
    // pq.processedCounts[0] = 1. pq.currentIndex should now point to P2 (index 0 of [2,1,0])

    pq.Enqueue(Task{ID: "D", Priority: 0, Index: 3}) // Add D(0) to P0 queue

    // P2 is empty. Dequeue should skip P2.
    // P1 is empty. Dequeue should skip P1.
    // pq.currentIndex should point to P0.
    // P0 queue has D.
	task = pq.Dequeue() // Should be D (P0)
	if task == nil || task.ID != "D" {
		t.Fatalf("Expected D, got %v. PQ state: curIdx %d, priorities %v, counts %v", formatTask(task), pq.currentIndex, pq.priorities, pq.processedCounts)
	}
}


// Helper to format task details for logging
func formatTask(task *Task) string {
	if task == nil {
		return "nil"
	}
	var sb strings.Builder
	sb.WriteString("ID:")
	sb.WriteString(task.ID)
	sb.WriteString(",Prio:")
	sb.WriteString(strconv.Itoa(task.Priority))
	return sb.String()
}

// Helper to format a slice of tasks for logging
func formatTasks(tasks []*Task) []string {
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		if task != nil {
			ids[i] = task.ID
		} else {
			ids[i] = "nil"
		}
	}
	return ids
}
