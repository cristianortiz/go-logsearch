package analyzer

import "testing"

func TestAnalyzerFileContent(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedErrors int
		expectedFile   string
	}{
		{
			name:           "Empty content",
			content:        "",
			expectedErrors: 0,
			expectedFile:   "",
		},
		{
			name:           "No errors found",
			content:        "INFO: System started\nINFO: User connected\nINFO: tasks completed",
			expectedErrors: 0,
			expectedFile:   "",
		},
		{
			name:           "Single error line",
			content:        "INFO: System starded\nERROR: connection failed\nINFO: tasks aborted",
			expectedErrors: 1,
			expectedFile:   "",
		},
		{
			name:           "Multiple error lines",
			content:        "ERROR: Starting error\nINFO: Re attempt\nERROR: second attemp failed\nERROR: System unnstable",
			expectedErrors: 3,
			expectedFile:   "",
		},
		{
			name:           "Case sensitivity test",
			content:        "error: must have all letters capitalized\nERROR: this will pass\nError: first letter capitalized only does'nt count\neRrOr: mixed does'nt count either",
			expectedErrors: 1,
			expectedFile:   "",
		},

		{
			name:           "ERROR at different positions",
			content:        "ERROR: At line start \nSome ERROR: in the middle\n  ERROR: start eith blank spaces",
			expectedErrors: 3,
			expectedFile:   "",
		},
		{
			name:           "Mixed content with line breaks",
			content:        "2023-01-01 10:00:00 INFO: Application started\n2023-01-01 10:05:00 ERROR: Connection fail\n2023-01-01 10:06:00 WARN: Re attempt\n2023-01-01 10:07:00 ERROR: Timeout during DB conn attempt\n2023-01-01 10:08:00 INFO: Connection renewed",
			expectedErrors: 2,
			expectedFile:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzeFileContent(tt.content)
			if result.ErrorCount != tt.expectedErrors {
				t.Errorf("AnalyzeFileContent() ErrorCount = %v,want %v", result.ErrorCount, tt.expectedErrors)
			}
			//check if returned type is correct
			if result.FileName != tt.expectedFile {
				t.Errorf("AnalyzeFileContent() FileName = %v, want %v", result.FileName, tt.expectedFile)
			}
		})

	}
}
