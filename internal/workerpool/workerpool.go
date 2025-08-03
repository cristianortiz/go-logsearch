package workerpool

import (
	"go-logsearch/internal/types"
	"sync"
)

// Tasks is an interfface to define the tasks that will be executed in the pool
type Task interface {
	Execute() (Result, error) //must containt the specifi logic assigned to a task
}

// Result is an interface that tasks must implement for their results
type Result interface {
	GetAnalysisResult() *types.AnalysisResult
	Err() error
}

type WorkerPoolBase interface {
	Submit(task Task)                      //sends a task to the pool to be processed
	Run()                                  //starts the pool creating and launching the workers
	Stop()                                 //stops the pool closing channels and waiting for workers to finish
	Results() <-chan types.ResultInterface // returns a reads only channel to receive tasks results
}

// WorkerPool is the struct that will be implementing the above interfaces, besides to have their own fields and methods
type WorkerPool struct {
	taskChan   chan Task
	resultChan chan types.ResultInterface
	numWorkers int
	wg         sync.WaitGroup
}

func NewWorkerPool(numWorkers int) WorkerPool {
	return WorkerPool{
		taskChan:   make(chan Task, 100),                  //buffer for tasks
		resultChan: make(chan types.ResultInterface, 100), //buffer for results
		numWorkers: numWorkers,
	}
}

func (p *WorkerPool) Submit(task Task) {
	p.taskChan <- task
}

func (p *WorkerPool) Run() {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *WorkerPool) Stop() {
	//indicates the workers there ir no more tasks to execute
	close(p.taskChan)
	//waits for all the workers to finish their tasks
	p.wg.Wait()
	close(p.resultChan)
}

func (p *WorkerPool) Results() <-chan types.ResultInterface {
	return p.resultChan

}

func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for task := range p.taskChan {
		result, err := task.Execute()
		p.resultChan <- result
		if err != nil {
			println("error doing the task", err)
		}
	}
}
