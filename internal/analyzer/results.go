package analyzer

import (
	"go-logsearch/internal/types"
)

type ReadFileResult struct {
	FilePath string
}

type ProcessChunkResult struct {
	Result types.AnalysisResult
}

func (r ReadFileResult) GetAnalysisResult() *types.AnalysisResult {
	return nil
}

func (r ReadFileResult) Err() error {
	return nil
}

func (r ProcessChunkResult) GetAnalysisResult() *types.AnalysisResult {
	return &r.Result

}

func (r ProcessChunkResult) Err() error {
	return nil
}
