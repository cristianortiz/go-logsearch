package analyzer

import (
	"go-logsearch/internal/workerpool"
	"os"
	"sync"

	"go.uber.org/zap"
)

// represents the content of a readed file
type FileContent struct {
	FileName string
	Content  string
}

// handles only reading files
type ReadFileTask struct {
	FilePath      string
	OutputChannel chan<- FileContent
	ReadFileWg    *sync.WaitGroup
}

type ProcessChunkTask struct {
	Content  string
	FileName string
}

func (r *ReadFileTask) Execute() (workerpool.Result, error) {
	log.Debug("Executing ReadFileTask", zap.String("file", r.FilePath))
	defer r.ReadFileWg.Done()
	content, err := os.ReadFile(r.FilePath)
	if err != nil {
		return ReadFileResult{}, err
	}
	//send content to outputChannel
	r.OutputChannel <- FileContent{
		FileName: r.FilePath,
		Content:  string(content),
	}
	return ReadFileResult{FilePath: r.FilePath}, nil

}

func (t ProcessChunkTask) Execute() (workerpool.Result, error) {
	log.Debug("Executing ProcessChunkTask")

	result := AnalyzeFileContent(t.Content)
	result.FileName = t.FileName
	log.Debug("ProcessChunkTask finished", zap.Int("error_count", result.ErrorCount))

	return ProcessChunkResult{Result: result}, nil
}
