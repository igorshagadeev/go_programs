package main

import (
    "fmt"
    "strings"
    "flag"
    "encoding/json"
    "os"
    "io"
    "io/ioutil"
    "log"
    "path/filepath"
    "bufio"
    "math"
    "runtime"
    "errors"
    "sync"
    "time"
    "strconv"
    "github.com/wcharczuk/go-chart"
)


// READ_BYTES and WORKERS can flexible control 
// memory and cpu consumption
const READ_BYTES = 100
const WORKERS = 2
const AGGREGATOR_BUFFER = 1000

var (
    root  string
    files []string
    err   error
)
var symbols = make(map[rune]int64)


func patcher(aggr <-chan rune) {
    // receive symbol <- aggr channel <- from workers
    // and safely patch symbols structure
    // due of single exclusive access
    //
    // finishes when aggr channel is closed
    for k := range aggr {
        _, found := symbols[k]
        if found {
            symbols[k] += 1
        } else {
            symbols[k] = 1
        }
    }
    fmt.Println("aggr finished")
}

// run workers could be detached from main func
// func run_workers(root string) {  }

func scanfile(paths <-chan string, aggr chan<- rune, done <-chan struct{}, wi int) {
    // worker that get path <- paths channel
    // until path channel is closed
    // until done channel is closed
    // open file at path and read READ_BYTES
    // to reduce memory consumption
    // send rune(byte) -> aggr channel
    fmt.Println("wi starts", wi)
    
    for path := range paths {
        fmt.Println("wi", wi, path)
        
        f, err := os.Open(path)
        if err != nil {
            log.Fatal(err)
        }
        defer func() {
            if err = f.Close(); err != nil {
                log.Fatal(err)
            }
        }()
        r := bufio.NewReader(f)
        b := make([]byte, READ_BYTES)
        for {
            _, err := r.Read(b)
            if err == io.EOF {
                break
            } else if err != nil {
                break
                fmt.Printf("error reading file %s", err)
            }
            
            for i := 0; i < READ_BYTES; i++ {
                k := []rune(string(b[i]))[0]

                select {
                    case aggr <- k:
                    case <-done:
                        return
                }
            }
        }
    }
}
    

func main() {
    
    t1 := time.Now()
    
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    flag.Parse()
    if len(flag.Args()) < 1 {
        fmt.Printf("usage: go run chan_worker_multiproc_read.go /path/to/dir\n")
        return
    }    
    root := flag.Args()[0]
    
    done := make(chan struct{})
    defer close(done)
    
    aggr := make(chan rune, AGGREGATOR_BUFFER)
    
    // filepath.Walk
    paths, errs := FilePathWalkDir(root, done)
    
    // start workers
    // a fixed number of goroutines WORKERS
    // to read and digest files.
    // until  done is closed.
    var wg sync.WaitGroup
    wg.Add(WORKERS)
    var i int
    for i = 0; i < WORKERS; i++ {
        go func(i int) {
            scanfile(paths, aggr, done, i)
            wg.Done()
        }(i)
    }
    // await finish and close aggr channel
    go func() {
        wg.Wait()
        close(aggr)
    }()
    
    // End of pipeline aggregate results
    patcher(aggr)
    
    // handle errs channel
    if err := <-errs; err != nil {
        panic(err)
    }
    
    // print info
    symbols_sum := int64(0)
    for _, v := range symbols{
        symbols_sum += v
    }
    fmt.Println("len symbols", len(symbols))
    fmt.Println("symbols amount ", symbols_sum)
    fmt.Println("symbols", symbols)
    
    t2 := time.Now()
    fmt.Println("read files in: ", t2.Sub(t1))
    
    // plot barchart map with ascii and frequency
    plot_barchart()
    // print symbols to file
    seed := strconv.Itoa(int(time.Now().Unix()))
    file, _ := json.MarshalIndent(symbols, "", " ")
    _ = ioutil.WriteFile("symbols_"+seed+".json", file, 0644)
    
    t3 := time.Now()
    fmt.Println("final: ", t3.Sub(t1))
}


func IsValidExt(ext string) bool {
    // element in array 
    switch ext {
        case
        "go",
        "txt",
        "js",
        "py":
        return true
    }
    return false
}

func FilePathWalkDir(root string, done <-chan struct{}) (<-chan string, <-chan error) {
    // make chans
    paths := make(chan string)
    errs := make(chan error, 1)
    
    // iterate over directory with files
    go func() {
        // Close the paths channel after Walk returns.
        defer close(paths)
        
        errs <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
            if !info.IsDir() {
                splitted := strings.Split(path, ".")
                if IsValidExt(splitted[len(splitted)-1]) {
                    //files = append(files, path)
                    // choose where to write
                    select {
                        case paths <- path:
                        case <-done:
                            return errors.New("walk canceled")
                    }
                }
            }
            return nil
        })
    }()
    return paths, errs
}

func plot_barchart(){
    // plot barchart to file
    
    bars := make([]chart.Value, 0, len(symbols))
    barslog := make([]chart.Value, 0, len(symbols))
    
    for k, v := range symbols {
        bars = append(bars, chart.Value{
            Value: float64(v), Label: string(k)} )
        barslog = append(barslog, chart.Value{
            Value: math.Log2(float64(v)), Label: string(k)} )
    }
    
    graph := chart.BarChart{
        Title: "Ascii frequency",
        Background: chart.Style{
            Padding: chart.Box{
                Top: 40,
                Right: 100,
            },
        },
        Height:   1000,
        Width:   2000,
        BarWidth: 2,
        Bars: bars,
    }
    
    f, _ := os.Create("output.png")
    defer f.Close()
    graph.Render(chart.PNG, f)
    
    graph2 := chart.BarChart{
        Title: "Ascii log2 frequency",
        Background: chart.Style{
            Padding: chart.Box{
                Top: 40,
                Right: 100,
            },
        },
        Height:   1000,
        Width:   2000,
        BarWidth: 2,
        Bars: barslog,
    }
    
    f2, _ := os.Create("output_log2.png")
    defer f2.Close()
    graph2.Render(chart.PNG, f2)
}
