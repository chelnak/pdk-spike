package validate

import (
	"fmt"
	"sync"

	"github.com/chelnak/ysmrr"
)

// Task encapsulates a work item that should go in a work
// pool.
type Task struct {
	// Err holds an error that occurred during a task. Its
	// result is only meaningful after Run has been called
	// for the pool that holds it.
	Err     error
	name    string
	Spinner *ysmrr.Spinner

	f func() error
}

// NewTask initializes a new task based on a given work
// function.
func NewTask(name string, f func() error) *Task {
	return &Task{name: name, f: f}
}

// Run runs a Task and does appropriate accounting via a
// given sync.WorkGroup.
func (t *Task) Run(wg *sync.WaitGroup) {
	t.Err = t.f()
	wg.Done()
}

// Pool is a worker group that runs a number of tasks at a
// configured concurrency.
type Pool struct {
	Tasks []*Task

	spinnerManager ysmrr.SpinnerManager
	concurrency    int
	tasksChan      chan *Task
	wg             sync.WaitGroup
}

// NewPool initializes a new pool with the given tasks and
// at the given concurrency.
func NewPool(tasks []*Task, concurrency int) *Pool {
	sm := ysmrr.NewSpinnerManager()
	for _, task := range tasks {
		spinner := sm.AddSpinner(task.name)
		task.Spinner = spinner
	}

	return &Pool{
		spinnerManager: sm,
		Tasks:          tasks,
		concurrency:    concurrency,
		tasksChan:      make(chan *Task),
	}
}

// Run runs all work within the pool and blocks until it's
// finished.
func (p *Pool) Run() {
	for i := 0; i < p.concurrency; i++ {
		go p.work()
	}

	p.spinnerManager.Start() // Remove when spinners can be started individually
	p.wg.Add(len(p.Tasks))
	for _, task := range p.Tasks {
		p.tasksChan <- task
	}

	// all workers return
	close(p.tasksChan)

	p.wg.Wait()
	p.spinnerManager.Stop()
}

// The work loop for any single goroutine.
func (p *Pool) work() {
	for task := range p.tasksChan {
		// Uncomment when this is implemented
		// task.Spinner.Start()
		task.Spinner.UpdateMessage(fmt.Sprintf("Validating with %s...", task.name))
		task.Run(&p.wg)
		if task.Err != nil {
			task.Spinner.Error()
		} else {
			task.Spinner.Complete()
		}
		task.Spinner.UpdateMessage(fmt.Sprintf("Validation with %s complete.", task.name))
	}
}

func CreateTask(name string, function func() error) *Task {
	return &Task{
		name: name,
		f:    function,
	}
}
