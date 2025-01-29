package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var serviceList map[string]string
var openColor = "\033[92m"
var closedColor = "\033[91m"
var resetColor = "\033[0m"

func loadServices() error {
	data, err := ioutil.ReadFile("services.json")
	if err != nil {
		return fmt.Errorf("error reading services file: %v", err)
	}
	serviceList = make(map[string]string)
	err = json.Unmarshal(data, &serviceList)
	if err != nil {
		return fmt.Errorf("error parsing services JSON: %v", err)
	}
	return nil
}

func getServiceName(port int) string {
	if service, exists := serviceList[strconv.Itoa(port)]; exists {
		return service
	}
	return "Unknown"
}

func detectService(host string, port int) string {
	target := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()

	service := getServiceName(port)
	if service != "Unknown" {
		return service
	}

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
	if service != "Unknown" {
		results <- fmt.Sprintf("%s[OPEN] Port %d is open - Service: %s%s", openColor, port, service, resetColor)
	}
}

func showHelp() {
	fmt.Println("Usage: <program> <host> <startPort> <endPort>")
	fmt.Println("\nFlags:")
	fmt.Println("-h   Show this help message.")
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "-h" {
		showHelp()
		os.Exit(0)
	}

	err := loadServices()
	if err != nil {
		fmt.Printf("Error loading services: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) != 4 {
		fmt.Println("Usage: <program> <host> <startPort> <endPort>")
		os.Exit(1)
	}
	host := os.Args[1]
	startPort, _ := strconv.Atoi(os.Args[2])
	endPort, _ := strconv.Atoi(os.Args[3])

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
