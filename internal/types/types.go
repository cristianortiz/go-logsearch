package types

// AnalysisResult represents the result of a the analysis of a file or a group of them
type AnalysisResult struct {
	FileName   string
	ErrorCount int // to count lines containt "ERROR" string
}

// ResultInterface define la interfaz común para resultados de tareas
type ResultInterface interface {
	GetAnalysisResult() *AnalysisResult
	Err() error
}

// FileChunk to process big files in smaller pieces
// type FileChunk struct {
// 	FileName string
// 	Content  []byte
// 	Offset   int
// }
