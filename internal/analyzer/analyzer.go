package analyzer

import (
	"bufio"
	"go-logsearch/internal/shared/logger"
	"os"
	"strings"

	"go.uber.org/zap"
)

// AnalyzeSingleFile reads and process a single file in a secuential way.
// returns the analysys result and a possible error
var log = logger.GetLogger()

func AnalyzeSingleFile(filepath string) (AnalysisResult, error) {
	result := AnalysisResult{
		FileName: filepath,
	}
	//open file
	file, err := os.Open(filepath)
	if err != nil {
		log.Error("error opening file",
			zap.String("file", filepath),
			zap.Error(err))
		return result, err

	}
	defer file.Close()

	//read file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		//simple processing logic, count lines with "ERROR" string
		if strings.Contains(line, "ERROR") {
			result.ErrorCount++
		}
		//adds more logic later
	}
	//handle errors during scanning
	if err := scanner.Err(); err != nil {
		log.Error("error scanning file",
			zap.String("file", filepath),
			zap.Error(err))
		return result, err
	}
	//return result and any error
	log.Info("File analyzed successfully", zap.String("file", filepath),
		zap.Int("error_count",
			result.ErrorCount))

	return result, nil
}

func AnalyzeFileContent(content string) AnalysisResult {
	result := AnalysisResult{}
	result.ErrorCount = 0
	scanner := strings.NewReader(content)
	bufScanner := bufio.NewScanner(scanner)
	for bufScanner.Scan() {
		line := bufScanner.Text()
		if strings.Contains(line, "ERROR") {
			result.ErrorCount++
		}
	}
	return result
}
