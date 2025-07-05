# Go Concurrent Log Search

This project aims to build a high-performance log file analysis tool using Go, leveraging the language's concurrency features. It utilizes a concurrent pipeline to efficiently process large volumes of data and provides a basic web interface to monitor the analysis progress. The idea is to first review the theoretical foundations of concurrency in Go and then apply them in the implementation of a real project.

## 📚 Foundations of Concurrency in Go

Go was designed with concurrency as a central pillar, offering powerful and simple tools like Goroutines and Channels, following the Communicating Sequential Processes (CSP) model.

### 1. Concurrency vs. Parallelism

- **Concurrency:** The ability to handle multiple tasks _apparently_ at the same time, by switching between them. In Go, this is achieved with goroutines multiplexed onto operating system threads by the runtime.
- **Parallelism:** The _simultaneous_ execution of multiple tasks on multi-core hardware. A concurrent Go program can achieve true parallelism if the runtime distributes it across available cores.

**Go Philosophy:** "Don't communicate by sharing memory; share memory by communicating."

### 2. Goroutines

- **Definition:** Functions or methods that execute concurrently. Lightweight and managed by the Go runtime.
- **Creation:** By prefixing a function call with `go`: `go myFunction()`.
- **Efficiency:** Minimal overhead, allowing thousands or millions to be created.
- **Scheduler:** The Go runtime efficiently manages goroutines on a smaller number of OS threads (M:N model).
- **Lifecycle:** They terminate when the function finishes. The program exits if the main goroutine (`main`) finishes.

### 3. Channels

- **Definition:** Typed conduits for sending and receiving values between goroutines safely.
- **Purpose:** Communication and synchronization, often avoiding the need for explicit locks.
- **Declaration:**
  - `make(chan Type)`: Unbuffered channel. Blocks sender until there is a receiver and vice versa (synchronization).
  - `make(chan Type, Capacity)`: Buffered channel. Blocks sending only if the buffer is full; blocks receiving only if the buffer is empty.
- **Operations:**
  - Sending: `ch <- value`
  - Receiving: `variable := <-ch`
  - Receiving with status: `value, ok := <-ch` (`ok` is false if the channel is closed and empty)
- **Closing Channels:** `close(ch)`. Indicates that no more values will be sent. **Important:** Close _from the sender_ only when no more sends will occur.
- **Iteration:** `for value := range ch { ... }` receives values until the channel is closed and empty.

### 4. The `select` Statement

- **Purpose:** To wait on multiple channel operations simultaneously.
- **Syntax:**
  ```go
  select {
  case <-chan1:
      // Action when chan1 is ready to receive
  case value := <-chan2:
      // Action when chan2 is ready, value received
  case chan3 <- anotherValue:
      // Action when chan3 is ready to send
  case <-time.After(timeout):
      // Timeout: no operation ready
  default:
      // No operation ready, execute immediately
  }
  ```
- **Behavior:** Chooses a ready operation randomly; blocks if none are ready (unless `default` is present).

### 5. Synchronization with the `sync` Package

Primitives for protecting direct shared memory (use with caution, prefer channels):

- **`sync.Mutex`:** Mutual exclusion lock (`Lock`/`Unlock`). Only one goroutine accesses at a time.
- **`sync.RWMutex`:** Reader/Writer lock (`RLock`/`RUnlock`/`Lock`/`Unlock`). Multiple concurrent readers, one exclusive writer.
- **`sync.WaitGroup`:** Wait for a collection of goroutines to finish. `Add(n)`, `Done()`, `Wait()`.
- **`sync.Once`:** Ensures a function is executed only once.
- **`sync.Pool`:** Pool of objects for reuse.

### 6. Context for Cancellation and Timeouts

- **`context` Package:** Managing cancellation signals, deadlines, and values across goroutine/API boundaries.
- `context.Context`: Interface that is passed explicitly.
- **Usage:** `context.WithCancel`, `context.WithTimeout`, `context.WithDeadline` create child contexts. The `cancel()` function stops goroutines that are "listening" on `<-ctx.Done()`.

### 7. Common Concurrency Patterns

(Visual diagrams could be included here for better understanding.)

- **Worker Pool:** A fixed number of goroutines processing tasks from an input channel. Controls concurrency.
- **Fan-out:** A source distributes work to multiple worker goroutines via a channel.
- **Fan-in:** Multiple worker goroutines send results to a single channel to be consolidated by an aggregator.
- **Pipelines:** A series of stages connected by channels. The output of one stage is the input to the next.
- **Confinement:** Ensuring that only one goroutine accesses a piece of data at a time, often implicitly with channels.
- **Error Handling:** Using dedicated error channels or packages like `errgroup`.

### 8. Best Practices and Common Pitfalls

- Communication vs. Synchronization: Channels for communication, Mutex for protecting shared state (if indispensable).
- Goroutine Leaks: Ensure goroutines have a clear exit condition (e.g., listening to context, closed channel).
- Closing Channels: Close only from the sender, only once.
- Detecting Closure: Use `val, ok := <-ch` or `for range`.
- `sync.WaitGroup`: Essential for waiting for groups of goroutines.
- Race Conditions: Use `go run -race` or `go test -race` to detect them.
- Context: Pass it to concurrent functions and I/O operations.

---

## Technical and Theoretical Foundations (Applied to the Project)

Let's revisit Go concurrency concepts, but now focusing on _how_ they will be specifically applied in this log analyzer project.

### 2.1. Goroutines: The Units of Work

- **Application:** Each concurrent task in the pipeline (discovery, individual file reading, chunk processing, aggregation, web server, context handling) will run as a separate goroutine.
- **Benefits:** They allow the program to perform multiple "tasks" seemingly at the same time, without needing to create heavy operating system threads. The Go runtime will efficiently manage the assignment of these goroutines to available threads.
- **Examples in the Project:**
  - `go discoverFiles(...)`: A goroutine that explores the directory.
  - `go fileReaderWorker(...)`: Multiple goroutines in a pool, each reading an assigned file.
  - `go dataProcessorWorker(...)`: Multiple goroutines in a pool, each processing a data chunk.
  - `go resultsAggregator(...)`: A goroutine that consolidates all results.
  - `go runWebServer(...)`: A goroutine that keeps the web server active.

### 2.2. Channels: Safe Communication

- **Application:** Channels will be the "pipes" connecting the different stages of the concurrent pipeline. They are the primary means for goroutines to pass data to each other and synchronize.
- **Channel Types in the Project:**
  - `chan string`: To pass file paths from the discoverer to the readers.
  - `chan FileChunk`: To pass read data chunks (e.g., `struct { FileName string; Content []byte }`) from the readers to the processors.
  - `chan AnalysisResult`: To pass partial results (e.g., `struct { FileName string; ErrorCount int; ImportantMessages []string }`) from the processors to the aggregator.
  - `chan error`: A dedicated channel for any goroutine to report critical errors that might require stopping the process.
  - `chan ControlSignal`: Possibly a channel for the web interface to send commands (start, stop) to the analysis process.
- **Buffers:** We will experiment with unbuffered channels (for strict synchronization) and buffered channels (to slightly decouple stages and allow some task/result accumulation).

### 2.3. `select`: Waiting on Multiple Events

- **Application:** Crucial for goroutines that need to react to multiple sources, such as receiving data from their input channel _or_ detecting that the context has been canceled.
- **Examples in the Project:**
  - Inside reader and processor workers:
    ```go
    select {
    case item, ok := <-inputCh:
        if !ok {
            // Input channel closed, exit
            return
        }
        // Process item
    case <-ctx.Done():
        // Context canceled, exit cleanly
        return
    }
    ```
  - In the main or control goroutine: Wait for the aggregator to finish, or for a fatal error to occur.

### 2.4. `sync.WaitGroup`: Synchronizing Groups of Goroutines

- **Application:** Indispensable for the main goroutine (`main`) or a coordinating goroutine to wait for all "child" goroutines (workers, discoverer, aggregator) to complete their execution before the main program exits.
- **Usage:**
  - `wg.Add(N)`: Increments the counter by the number of goroutines to be launched.
  - `defer wg.Done()`: Called at the end of each launched goroutine, decrements the counter.
  - `wg.Wait()`: Blocks until the counter reaches zero.

### 2.5. `context.Context`: Cancellation and Timeout Management

- **Application:** The primary mechanism for propagating cancellation signals throughout the concurrent pipeline. If the web interface requests to stop the analysis, or if a serious error occurs, we will cancel the context.
- **Usage:** A main context (e.g., `context.WithCancel`) will be created when starting the analysis process. This context will be passed as the first argument to _all_ functions that launch/execute goroutines within the pipeline. Inside the goroutines, `<-ctx.Done()` will be monitored to react to cancellation.

### 2.6. Key Concurrency Patterns

- **Worker Pool:** Implemented for the reading and processing stages. A fixed number of goroutines (`N` readers, `M` processors) will wait on input channels, take a task, process it, and send the result to an output channel. This limits concurrent resource consumption (CPU, I/O).
- **Fan-out:** The file discovery goroutine will feed the path channel, which will be read by multiple reader goroutines (the reader worker pool). The work (list of files) is "fanned out" across several workers.
- **Fan-in:** The processor worker pool goroutines will write their partial results to a single results channel. A single goroutine (the aggregator) will read from this channel, "fanning in" all the results.
- **Pipelines:** The overall data flow structure (Discovery -> Reading -> Processing -> Aggregation) will form a pipeline. Each stage reads from one channel, performs work, and writes to another channel.
- **Confinement:** Ensuring that only one goroutine accesses a piece of data at a time, often implicitly with channels.
- **Error Handling:** Using dedicated error channels or packages like `errgroup`.
