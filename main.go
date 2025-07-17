package main

import (
	"fmt"
	"go-logsearch/internal/analyzer"
	"go-logsearch/internal/shared/logger"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

const (
	logDir     = "logs"
	numWorkers = 4 //concurrents workers to analyze files
)

func main() {

	log := logger.GetLogger()
	defer log.Sync()
	log.Info("-- Starting CONCURRENT log files analysis --", zap.Int("workers", numWorkers))

	//-- Channels to goroutines communications --
	//to send file routes from discoverer to workers, the buffer is used to avoid blocking
	// the discoverer if workers are busy
	jobs := make(chan string, 10)
	//to send results from workers to "agreggator"
	results := make(chan analyzer.AnalysisResult, 10)

	var wg sync.WaitGroup

	// launch workers (consumers)
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1) // Incrementar el contador del WaitGroup
		go worker(i, &wg, jobs, results)
	}

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
