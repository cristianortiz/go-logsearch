package analyzer

// AnalysisResult represents the result of a the analysis of a file or a group of them
type AnalysisResult struct {
	FileName   string
	ErrorCount int // to count lines containt "ERROR" string
}

// FileChunk to process big files in smaller pieces
// type FileChunk struct {
// 	FileName string
// 	Content  []byte
// 	Offset   int
// }
