package main

import (
    "fmt"
    "net"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"
    "bufio"
    "flag"
)


var (
    openColor   = "\033[92m"
    closedColor = "\033[91m"
    resetColor  = "\033[0m"
)


func detectService(host string, port int) string {
    target := fmt.Sprintf("%s:%d", host, port)
    conn, err := net.DialTimeout("tcp", target, 2*time.Second)
    if err != nil {
        return ""
    }
    defer conn.Close()

    conn.SetDeadline(time.Now().Add(2 * time.Second))
    reader := bufio.NewReader(conn)
    banner, _ := reader.ReadString('\n')

    banner = strings.TrimSpace(banner)
    if banner != "" {
        return fmt.Sprintf("Custom: %s", banner)
    }

    return "Unknown"
}


func scanPort(host string, port int, wg *sync.WaitGroup, results chan<- string) {
    defer wg.Done()

    target := fmt.Sprintf("%s:%d", host, port)
    conn, err := net.DialTimeout("tcp", target, 2*time.Second)
    if err != nil {
        results <- fmt.Sprintf("%s[CLOSED] Port %d is closed%s", closedColor, port, resetColor)
        return
    }

    conn.Close()
    service := detectService(host, port)
    if service != "" {
        results <- fmt.Sprintf("%s[OPEN] Port %d is open - Service: %s%s", openColor, port, service, resetColor)
    } else {
        results <- fmt.Sprintf("%s[OPEN] Port %d is open%s", openColor, port, resetColor)
    }
}


func printHelp() {
    fmt.Println("Usage: <program> <host> <startPort> <endPort>")
    fmt.Println("Options:")
    fmt.Println("-h : Show this help message")
}


func main() {
    help := flag.Bool("h", false, "Show help message")
    flag.Parse()

    if *help {
        printHelp()
        os.Exit(0)
    }

    if len(flag.Args()) != 3 {
        printHelp()
        os.Exit(1)
    }

    host := flag.Args()[0]
    startPort, _ := strconv.Atoi(flag.Args()[1])
    endPort, _ := strconv.Atoi(flag.Args()[2])

    var wg sync.WaitGroup
    results := make(chan string, endPort-startPort+1)
    fmt.Printf("Starting port scan on host %s...\n", host)

    if net.ParseIP(host) == nil {
        _, err := net.LookupHost(host)
        if err != nil {
            fmt.Printf("Invalid IP address or hostname: %s\n", host)
            os.Exit(1)
        }
    }

    for port := startPort; port <= endPort; port++ {
        wg.Add(1)
        go scanPort(host, port, &wg, results)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    for result := range results {
        fmt.Println(result)
    }
}
