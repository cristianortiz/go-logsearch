package main

import "go-logsearch/internal/shared/logger"

func main() {

	log := logger.GetLogger()
	defer log.SyncLogger()
}
