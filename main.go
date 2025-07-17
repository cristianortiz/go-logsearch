package main

import (
	"fmt"
	"go-logsearch/internal/analyzer"
	"go-logsearch/internal/shared/logger"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func main() {

	log := logger.GetLogger()
	defer log.Sync()
	log.Info("-- Starting sequential log files analysis --")

	//log file directory
	logDir := "logs"

	files, err := os.ReadDir(logDir)
	if err != nil {
		log.Fatal("logs directory can't be read it..",
			zap.String("directory", logDir),
			zap.Error(err))
	}

	var totalErrorCount int
	var filesAnalyzed int
	// analyzing every log file
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".log" {
			filePath := filepath.Join(logDir, file.Name())
			log.Info("Analyzing file", zap.String("file", filePath))

			result, err := analyzer.AnalyzeSingleFile(filePath)
			if err != nil {
				log.Warn("file ommited due error analysis",
					zap.String("file", filePath))
				continue
			}
			filesAnalyzed++
			totalErrorCount += result.ErrorCount
		}
	}

	//print results
	log.Info("--- Análisis secuencial completado ---",
		zap.Int("files_analyzed", filesAnalyzed),
		zap.Int("total_errors_found", totalErrorCount),
	)

	fmt.Println("\n======================================")
	fmt.Printf(" analysis resume\n")
	fmt.Println("======================================")
	fmt.Printf(" Analyzed Files: %d\n", filesAnalyzed)
	fmt.Printf(" Total ERROR lines founded: %d\n", totalErrorCount)
	fmt.Println("======================================")

}
