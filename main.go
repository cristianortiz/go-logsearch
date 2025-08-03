package main

import (
	"context"
	"fmt"
	"go-logsearch/internal/analyzer"
	"go-logsearch/internal/shared/logger"
	"go-logsearch/internal/workerpool"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer log.Sync()
	log.Info("-- Starting CONCURRENT (Worker Pools - Fans) log files analysis --",
		zap.Int("reader_workers", numReaderWorkers),
		zap.Int("processor_workers", numProcessorWorkers))

	//-- Channels to goroutines communications --
	//channel for file routes
	filePaths := make(chan string, 10)
	//-- WorkerPools--
	readerPool := workerpool.NewWorkerPool(numReaderWorkers)
	processorPool := workerpool.NewWorkerPool(numReaderWorkers)
	//-- Launch workerpools
	go readerPool.Run(ctx)
	go processorPool.Run(ctx)

	// launch goroutines to monitoring readerPool and processorPool errors
	go func() {
		for err := range readerPool.Errors() {
			log.Error("readPool error detected", zap.Error(err))
			// critical error cancel ops
			if isCriticalError(err) {
				log.Error("critical cancel operation")
				cancel()
				return
			}
		}
	}()
	go func() {
		for err := range processorPool.Errors() {
			log.Error("processorPool error detected", zap.Error(err))
			if isCriticalError(err) {
				log.Error("critical cancel operation")
				cancel()
				return
			}
		}
	}()

	//--waitGroups to sync
	var discoverWg sync.WaitGroup
	var readFileWg sync.WaitGroup
	var processingWg sync.WaitGroup

	discoverWg.Add(1)
	go discoverFiles(&discoverWg, filePaths)

	// Procesar archivos (consumer)
	processingWg.Add(1)
	go processFiles(&processingWg, &readFileWg, filePaths, &readerPool, &processorPool)

	// Esperar a que termine el procesamiento
	processingWg.Wait()

	// Recopilar y mostrar resultados
	collectAndDisplayResults(&processorPool)

}

// discoverFiles search files with .log extension inside directory and send it to jobs channel
func discoverFiles(wg *sync.WaitGroup, filePaths chan<- string) {
	log.Debug("Entering discoverfiles goroutine")

	defer wg.Done()
	defer close(filePaths)

	err := filepath.WalkDir(logDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".log" {
			log.Debug("File founded, sending to jobs", zap.String("file", path))
			filePaths <- path
		}
		return nil
	})

	if err != nil {
		log.Error("Error searching for files", zap.Error(err))
	}

}

func processFiles(processingWg *sync.WaitGroup, readFileWg *sync.WaitGroup,
	filePaths <-chan string, readerPool *workerpool.WorkerPool,
	processorPool *workerpool.WorkerPool) {

	defer processingWg.Done()
	log.Debug("Entering processFiles goroutine")

	for filePath := range filePaths {
		readFileWg.Add(1)
		log.Debug("Submitting ReadFileTask", zap.String("file", filePath))

		//creates and send read task
		readFileTask := &analyzer.ReadFileTask{
			FilePath:      filePath,
			ProcessorPool: processorPool,
			ReadFileWg:    readFileWg,
		}
		readerPool.Submit(readFileTask)
	}
	// waits for all readers task are finished
	readFileWg.Wait()
	log.Info("All read tasks completed, stopping readerPool")
	readerPool.Stop()
	//waits for all chunks are processed

	processorPool.Stop()
}

func collectAndDisplayResults(processorPool *workerpool.WorkerPool) {
	var totalErrorCount int
	var filesAnalyzed int

	for r := range processorPool.Results() {
		if r.Err() != nil {
			log.Error("Error processing chunk", zap.Error(r.Err()))
			continue
		}
		// Obtener el resultado usando el nuevo método específico
		analysisResult := r.GetAnalysisResult()
		if analysisResult != nil {
			totalErrorCount += analysisResult.ErrorCount
			filesAnalyzed++
			log.Debug("Processed file",
				zap.Int("errors", analysisResult.ErrorCount),
				zap.Int("total_so_far", totalErrorCount))
		}
	}

	// Mostrar resultados
	log.Info("-- Concurrent Analysis finished --",
		zap.Int("files_analyzed", filesAnalyzed),
		zap.Int("total_errors_found", totalErrorCount),
	)

	fmt.Println("\n======================================")
	fmt.Printf(" Concurrent Analysis resume\n")
	fmt.Println("======================================")
	fmt.Printf(" Reader Workers used: %d\n", numReaderWorkers)
	fmt.Printf(" Processor Workers used: %d\n", numProcessorWorkers)
	fmt.Printf(" Archivos analizados: %d\n", filesAnalyzed)
	fmt.Printf(" Total 'ERROR'lines founded: %d\n", totalErrorCount)
	fmt.Println("======================================")
}

func isCriticalError(err error) bool {

	// logic to define if an error is critical for the app operation  ex. directory or file access, disc related errors, etc
	return false
}
