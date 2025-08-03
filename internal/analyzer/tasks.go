package analyzer

import (
	"go-logsearch/internal/workerpool"
	"os"
	"sync"

	"go.uber.org/zap"
)

type ReadFileTask struct {
	FilePath      string
	ProcessorPool *workerpool.WorkerPool
	ReadFileWg    *sync.WaitGroup
}

type ProcessChunkTask struct {
	Content string
}

func (r *ReadFileTask) Execute() (workerpool.Result, error) {
	log.Debug("Executing ReadFileTask", zap.String("file", r.FilePath))
	defer r.ReadFileWg.Done()
	content, err := os.ReadFile(r.FilePath)
	if err != nil {
		return ReadFileResult{}, err
	}
	//send processChunkTask to processorPool
	r.ProcessorPool.Submit(ProcessChunkTask{Content: string(content)})
	return ReadFileResult{}, nil

}

func (t ProcessChunkTask) Execute() (workerpool.Result, error) {
	log.Debug("Executing ProcessChunkTask")
	result := AnalyzeFileContent(t.Content)
	log.Debug("ProcessChunkTask finished", zap.Int("error_count", result.ErrorCount))
	return ProcessChunkResult{Result: result}, nil
}
