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
	logDir              = "logs"
	numReaderWorkers    = 2 //concurrents workers to only reads files
	numProcessorWorkers = 2 // concurrents workers to process files
)

var log = logger.GetLogger()

func main() {

	defer log.Sync()
	log.Info("-- Starting CONCURRENT (Worker Pools - Fans) log files analysis --", zap.Int("reader_workers", numReaderWorkers),
		zap.Int("processor_workers", numProcessorWorkers))

	//-- Channels to goroutines communications --
	//to send file routes from discoverer to workers, the buffer is used to avoid the discoverer is blocked, if workers are busy
	jobs := make(chan string, 10)
	// to send files content from reader_workers to processor_workers
	dataChunks := make(chan string, 10)
	//to send results from procesor_workers to "agreggator"in this case main goroutine
	results := make(chan analyzer.AnalysisResult, 10)
	// main WG, for waiting to all goroutines to finish their work
	var mainWg sync.WaitGroup

	//Launch Discoverer (productor)
	mainWg.Add(1)
	go discoverFiles(&mainWg, jobs)

	// launch readers_workers (consumers) with their own WG, besides the mainWg
	var readerWg sync.WaitGroup
	for i := 1; i <= numReaderWorkers; i++ {
		readerWg.Add(1)
		mainWg.Add(1)
		go fileReaderWorker(i, &mainWg, &readerWg, jobs, dataChunks)
	}

	// Goroutine to close dataChunks channel when reader_workers ends
	go func() {
		readerWg.Wait()
		close(dataChunks)
		log.Info("dataChunks channel closed")

	}()

	// launching processor workers
	var processorWg sync.WaitGroup
	for i := 1; i <= numProcessorWorkers; i++ {
		processorWg.Add(1)
		mainWg.Add(1)
		go dataProcessorWorker(i, &mainWg, &processorWg, dataChunks, results)
	}

	// Goroutine to wait for all the workers are finished, and then closes the results channel
	go func() {
		mainWg.Wait()
		close(results)
		log.Info("results channel closed")

	}()

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
	fmt.Printf(" Concurrent Analysis resume\n")
	fmt.Println("======================================")
	fmt.Printf(" Analyzed Files: %d\n", filesAnalyzed)
	fmt.Printf(" Total ERROR lines founded: %d\n", totalErrorCount)
	fmt.Println("======================================")

}

// discoverFiles search files with .log extension inside directory and send it to jobs channel
func discoverFiles(wg *sync.WaitGroup, jobs chan<- string) {
	defer wg.Done()
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

// fileReaderWorker i a goroutine thats receive file routes from jobs channel, read it and send their content to dataChunks chan
func fileReaderWorker(id int, mainWg *sync.WaitGroup, workerWg *sync.WaitGroup, jobs <-chan string, dataChunks chan<- string) {

	defer mainWg.Done()
	defer workerWg.Done()
	log.Info("Reader worker started", zap.Int("worker_id", id))

	for filePath := range jobs {
		log.Debug("Reader worker is reading a file", zap.Int("worker_id", id), zap.String("file", filePath))
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Error("error at reading file", zap.Int("worker_id", id),
				zap.String("file", filePath),
				zap.Error(err))
		}
		dataChunks <- string(content)
	}
	log.Info("Reader worker finished, no more job.", zap.Int("worker_id", id))
}

// dataProcessorWorker is a goroutine thats receives file's content from dataChunks channel, process it
// and send the results to results channel
func dataProcessorWorker(id int, mainWg *sync.WaitGroup, workerWg *sync.WaitGroup, dataChunks <-chan string, results chan<- analyzer.AnalysisResult) {
	defer mainWg.Done()
	defer workerWg.Done()

	log.Info("Processor worker started", zap.Int("worker_id", id))

	for content := range dataChunks {

		log.Debug("Processor worker processing data chunk", zap.Int("worker_id", id))
		result := analyzer.AnalyzeFileContent(content)

		results <- result
	}

	log.Info("Processor worker finished, no more dataChunks.", zap.Int("worker_id", id))

}
