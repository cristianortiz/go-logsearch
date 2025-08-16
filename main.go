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
	filePaths := make(chan string, 10)                  //step 1 -> step 2
	fileContents := make(chan analyzer.FileContent, 10) //step 2 -> step 3

	//-- WorkerPools--
	readerPool := workerpool.NewWorkerPool(numReaderWorkers)
	processorPool := workerpool.NewWorkerPool(numReaderWorkers)

	//-- Launch workerpools
	go readerPool.Run(ctx)
	go processorPool.Run(ctx)

	//monitoring errors
	errorMonitor(cancel, &readerPool, &processorPool)

	//--waitGroups to sync
	var discoverWg sync.WaitGroup
	var readFileWg sync.WaitGroup
	var processingWg sync.WaitGroup

	//--step 1 discover files
	discoverWg.Add(1)
	go discoverFiles(ctx, &discoverWg, filePaths)

	//--step 2 read files
	readFileWg.Add(1)
	go readFiles(ctx, &readFileWg, filePaths, fileContents, &readerPool)

	//--step 3 process content
	processingWg.Add(1)
	go processContents(ctx, &processingWg, fileContents, &processorPool)
	processingWg.Wait()

	collectAndDisplayResults(&processorPool)

}

// discoverFiles search files with .log extension inside directory and send it to jobs channel
func discoverFiles(ctx context.Context, wg *sync.WaitGroup, filePaths chan<- string) {
	log.Debug("Entering discoverfiles goroutine")

	defer wg.Done()
	defer close(filePaths)

	err := filepath.WalkDir(logDir, func(path string, info os.DirEntry, err error) error {
		select {
		//check cancelation signal in everty iteration
		case <-ctx.Done():
			return ctx.Err() //stop the walkdir if a context cancelation signal was received

		//continue normal proccesing
		default:
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".log" {
				log.Debug("File found, sending to channel", zap.String("file", path))
				//send to channel but also check for cancelation signal
				select {
				case filePaths <- path: //file sended correctly
					//context canceled while sending is executed
				case <-ctx.Done():
					return ctx.Err()

				}
			}
			return nil
		}
	})
	//check if the err is 'normal', about the walkdir logic only, a context signal cancelation is not an err per se
	if err != nil && err != context.Canceled {
		log.Error("Error searching for files", zap.Error(err))
	}

	log.Info("File discovery completed")
}

func readFiles(ctx context.Context, wg *sync.WaitGroup, filePaths <-chan string, fileContents chan<- analyzer.FileContent, readerPool *workerpool.WorkerPool) {
	defer wg.Done()
	log.Debug("Starting file reading stage")

	var tasksWg sync.WaitGroup
	//process every file path
	for filePath := range filePaths {
		select {
		case <-ctx.Done():
			return
		default:
			tasksWg.Add(1)
			log.Debug("Submitting ReadFileTask", zap.String("file", filePath))
			//creates and send reads tasks
			readFileTask := &analyzer.ReadFileTask{
				FilePath:      filePath,
				OutputChannel: fileContents,
				ReadFileWg:    &tasksWg,
			}
			readerPool.Submit(readFileTask)
		}
	}

	//waits for all reader task are finished
	tasksWg.Wait()
	//close channel qhen all reads are done
	close(fileContents)
	log.Info("All files have need read")
	readerPool.Stop()

}

func processContents(ctx context.Context, wg *sync.WaitGroup, fileContents <-chan analyzer.FileContent, processorPool *workerpool.WorkerPool) {
	defer wg.Done()
	log.Debug("Starting content processing stage")

	//process every content fragment
	for content := range fileContents {
		select {
		case <-ctx.Done():
			return
		default:
			log.Debug("Submittin ProcessChunkTask", zap.String("file", content.FileName))
			//creates and send process tasks
			processTask := analyzer.ProcessChunkTask{
				Content:  content.Content,
				FileName: content.FileName,
			}
			processorPool.Submit(processTask)
		}
	}
	log.Info("All content has been processed")
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
		analysisResult := r.GetAnalysisResult()
		if analysisResult != nil {
			totalErrorCount += analysisResult.ErrorCount
			filesAnalyzed++
			log.Debug("Processed file",
				zap.String("file", analysisResult.FileName),
				zap.Int("errors", analysisResult.ErrorCount),
				zap.Int("total_so_far", totalErrorCount))
		}
	}

	// show results
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

func errorMonitor(cancel context.CancelFunc, readerPool, processorPool *workerpool.WorkerPool) {
	// readerPool monitoring errors
	go func() {
		for err := range readerPool.Errors() {
			log.Error("ReaderPool error", zap.Error(err))
			if isCriticalError(err) {
				log.Error("Critical error in reader pool, cancelling operation")
				cancel()
				return
			}
		}
	}()
	// processorPool monitoring errors
	go func() {
		for err := range processorPool.Errors() {
			log.Error("ProcessorPool error", zap.Error(err))
			if isCriticalError(err) {
				log.Error("Critical error in processor pool, cancelling operation")
				cancel()
				return
			}
		}
	}()
}

func isCriticalError(err error) bool {

	// logic to define if an error is critical for the app operation  ex. directory or file access, disc related errors, etc
	// for now just log the error
	log.Error("critital error", zap.Error(err))
	return false
}
