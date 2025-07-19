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

var log = logger.GetLogger()

func main() {

	defer log.Sync()
	log.Info("-- Starting CONCURRENT log files analysis --", zap.Int("workers", numWorkers))

	//-- Channels to goroutines communications --
	//to send file routes from discoverer to workers, the buffer is used to avoid blocking
	// the discoverer if workers are busy
	jobs := make(chan string, 10)
	//to send results from workers to "agreggator"
	results := make(chan analyzer.AnalysisResult, 10)

	var wg sync.WaitGroup
	//producer
	go discoverFiles(jobs)

	// goroutine for wait for all workers are finished, then closes 'results' channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// launch workers (consumers)
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, &wg, jobs, results)
	}

	// aggregate or collect results from 'result' channel
	var totalErrorCount int
	var filesAnalyzed int
	for result := range results {
		log.Info("Result received",
			zap.String("file", result.FileName),
			zap.Int("errors_found", result.ErrorCount),
		)
		filesAnalyzed++
		totalErrorCount += result.ErrorCount
	}

	//print results
	log.Info("-- Concurrent Analysis finished --",
		zap.Int("files_analyzed", filesAnalyzed),
		zap.Int("total_errors_found", totalErrorCount),
	)

	fmt.Println("\n======================================")
	fmt.Printf(" concurrent analysis resume\n")
	fmt.Println("======================================")
	fmt.Printf(" Analyzed Files: %d\n", filesAnalyzed)
	fmt.Printf(" Total ERROR lines founded: %d\n", totalErrorCount)
	fmt.Println("======================================")

}

// discoverFiles search files with .log extension inside directory and send it to jobs channel
func discoverFiles(jobs chan<- string) {
	defer close(jobs)
	err := filepath.WalkDir(logDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".log" {
			log.Debug("File founded, sending to jobs", zap.String("file", path))
			jobs <- path
		}
		return nil
	})

	if err != nil {
		log.Error("Error searching for files", zap.Error(err))
	}
}

// worker is a goroutine that receive file routes from 'jobs channel'
// process it and send results to 'result' channel
func worker(id int, wg *sync.WaitGroup, jobs <-chan string, results chan<- analyzer.AnalysisResult) {
	defer wg.Done()
	log.Info("worker started", zap.Int("worker_id", id))

	//the loop executes on a channel until the channel is closed and get empty
	for filePath := range jobs {
		log.Debug("Worker processing file", zap.Int("worker_id", id), zap.String("file", filePath))
		result, err := analyzer.AnalyzeSingleFile(filePath)
		if err != nil {
			log.Warn("Worker has ommited file on error",
				zap.Int("worker_id", id),
				zap.String("file", filePath),
				zap.Error(err),
			)
			continue
		}
		results <- result
	}
	log.Info("Worker finished, no more jobs to process", zap.Int("worker_id", id))

}
