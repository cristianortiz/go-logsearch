package workerpool

import (
	"context"
	"go-logsearch/internal/shared/logger"
	"go-logsearch/internal/types"
	"sync"

	"go.uber.org/zap"
)

var log = logger.GetLogger()

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
	Run(ctx context.Context)               //starts the pool creating and launching the workers
	Stop()                                 //stops the pool closing channels and waiting for workers to finish
	Results() <-chan types.ResultInterface // returns a reads only channel to receive tasks results
	Errors() <-chan error
}

// WorkerPool is the struct that will be implementing the above interfaces, besides to have their own fields and methods
type WorkerPool struct {
	taskChan   chan Task
	resultChan chan types.ResultInterface
	errorChan  chan error
	numWorkers int
	wg         sync.WaitGroup
}

func NewWorkerPool(numWorkers int) WorkerPool {
	return WorkerPool{
		taskChan:   make(chan Task, 100),                  //buffer for tasks
		resultChan: make(chan types.ResultInterface, 100), //buffer for results
		errorChan:  make(chan error, numWorkers),          //with buffer to avoid blocks
		numWorkers: numWorkers,
	}
}

func (p *WorkerPool) Submit(task Task) {
	p.taskChan <- task
}

func (p *WorkerPool) Run(ctx context.Context) {
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *WorkerPool) Stop() {
	//indicates the workers there ir no more tasks to execute
	close(p.taskChan)
	//waits for all the workers to finish their tasks
	p.wg.Wait()
	close(p.resultChan)
	close(p.errorChan)
}

func (p *WorkerPool) Results() <-chan types.ResultInterface {
	return p.resultChan

}

func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			//context canceled clean exit
			return
		case task, ok := <-p.taskChan:
			if !ok {
				//closed channel no mor tasks to complete
				return
			}
			//execute the task
			result, err := task.Execute()
			if err != nil {
				//try to send the error  without blocking the channel if it's full
				select {
				//error sended correctly
				case p.errorChan <- err:

				default:
					//channel full, logging the error localy
					log.Error("error executing task", zap.Error(err))
				}
				continue
			}

			//send results, considering context cancelation
			select {
			case p.resultChan <- result:
			case <-ctx.Done():
				//context canceled while sending results
			}
		}
	}
}

func (p *WorkerPool) Errors() <-chan error {
	return p.errorChan
}
