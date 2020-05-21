package main

import (
    "fmt"
    "io"
    "encoding/json"
    "io/ioutil"
    "strings"
    "os"
    "log"
    "path/filepath"
    "bufio"
    "math"
    "flag"
    "runtime"
    "time"
    "sync"
    "strconv"
    "github.com/wcharczuk/go-chart"
)


const READ_BYTES = 100
var (
    root  string
    files []string
    err   error
)
var symbols = make(map[rune]int64)
var aMutex sync.Mutex



func main() {
    
    t1 := time.Now()
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    flag.Parse()
    if len(flag.Args()) < 1 {
        fmt.Printf("usage: go run mutex_multiproc_read_files.go /path/to/dir\n")
        return
    }
    root := flag.Args()[0]
    
    // filepath.Walk
    files, err = FilePathWalkDir(root)
    if err != nil {
        panic(err)
    }
    
    var wg sync.WaitGroup
    var i int
    
    for i = 0; i < len(files); i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            scanfile(files[i])
        }(i)
    }
    wg.Wait()
    
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

func FilePathWalkDir(root string) ([]string, error) {
    // iterate over directory with files
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if !info.IsDir() {
            splitted := strings.Split(path, ".")
            if IsValidExt(splitted[len(splitted)-1]) {
                files = append(files, path)
            }
        }
        return nil
    })
    return files, err
}


func scanfile(path string) {
    // read chunk READ_BYTES from file
    // scan symbol by symbol
    // and securely update map of ascii
    
    fmt.Println("path", path)
    
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
        //lock before dangerous operation
        aMutex.Lock()
        // dangerous operation
        for i := 0; i < READ_BYTES; i++ {
            k := []rune(string(b[i]))[0]
            _, found := symbols[k]
            if found {
                symbols[k] += 1
            } else {
                symbols[k] = 1
            }
        }
        aMutex.Unlock()
    }
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
