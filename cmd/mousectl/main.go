package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type resultResponse struct {
	OK         bool   `json:"ok"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error"`
}

type searchResponse struct {
	Matches []struct {
		Path    string  `json:"path"`
		Score   float64 `json:"score"`
		Snippet string  `json:"snippet"`
	} `json:"matches"`
}

type logEntry struct {
	Timestamp string            `json:"ts"`
	Level     string            `json:"level"`
	Service   string            `json:"service"`
	Message   string            `json:"msg"`
	Fields    map[string]string `json:"fields"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "status":
		statusCmd(os.Args[2:])
	case "run":
		runCmd(os.Args[2:])
	case "reindex":
		reindexCmd(os.Args[2:])
	case "search":
		searchCmd(os.Args[2:])
	case "approve":
		approveCmd(os.Args[2:])
	case "logs":
		logsCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "mousectl <status|run|reindex|search|approve|logs>")
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	addr := fs.String("addr", "http://localhost:8080", "gateway address")
	_ = fs.Parse(args)

	resp, err := http.Get(strings.TrimRight(*addr, "/") + "/health")
	if err != nil {
		fmt.Fprintf(os.Stderr, "status error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "status error: http %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	addr := fs.String("addr", "http://localhost:8080", "gateway address")
	tool := fs.String("tool", "exec", "tool name")
	_ = fs.Parse(args)
	command := fs.Args()
	if len(command) == 0 {
		fmt.Fprintln(os.Stderr, "run requires command")
		os.Exit(2)
	}
	payload := map[string]any{"tool": *tool, "command": command}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(*addr, "/") + "/tools/run"
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "run error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var parsed resultResponse
	_ = json.Unmarshal(data, &parsed)
	if resp.StatusCode != http.StatusOK || !parsed.OK {
		fmt.Fprintf(os.Stderr, "run failed: %s\n", parsed.Error)
		if parsed.Stderr != "" {
			fmt.Fprintln(os.Stderr, parsed.Stderr)
		}
		os.Exit(1)
	}
	if parsed.Stdout != "" {
		fmt.Print(parsed.Stdout)
	}
}

func reindexCmd(args []string) {
	fs := flag.NewFlagSet("reindex", flag.ExitOnError)
	addr := fs.String("addr", "http://localhost:8080", "gateway address")
	_ = fs.Parse(args)
	endpoint := strings.TrimRight(*addr, "/") + "/index/reindex"
	req, err := http.NewRequest(http.MethodPost, endpoint, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reindex error: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reindex error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "reindex failed: %s\n", string(data))
		os.Exit(1)
	}
	fmt.Println("ok")
}

func searchCmd(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	addr := fs.String("addr", "http://localhost:8080", "gateway address")
	query := fs.String("q", "", "search query")
	limit := fs.Int("limit", 5, "max results")
	_ = fs.Parse(args)
	if *query == "" {
		fmt.Fprintln(os.Stderr, "search requires -q query")
		os.Exit(2)
	}
	endpoint := fmt.Sprintf("%s/index/search?q=%s&limit=%d", strings.TrimRight(*addr, "/"), url.QueryEscape(*query), *limit)
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "search failed: %s\n", string(data))
		os.Exit(1)
	}
	var parsed searchResponse
	_ = json.Unmarshal(data, &parsed)
	for _, match := range parsed.Matches {
		fmt.Printf("%0.2f %s\n", match.Score, match.Path)
		if match.Snippet != "" {
			fmt.Println(match.Snippet)
		}
	}
}

func approveCmd(args []string) {
	fs := flag.NewFlagSet("approve", flag.ExitOnError)
	addr := fs.String("addr", "http://localhost:8080", "gateway address")
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "approve requires id")
		os.Exit(2)
	}
	payload := map[string]string{"id": fs.Arg(0)}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(*addr, "/") + "/approvals/submit"
	resp, err := http.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "approve error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "approve failed: %s\n", string(data))
		os.Exit(1)
	}
	fmt.Println("ok")
}

func logsCmd(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	path := fs.String("file", "./runtime/logs/mouse.log", "log file path")
	lines := fs.Int("n", 50, "lines to show")
	_ = fs.Parse(args)

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logs error: %v\n", err)
		os.Exit(1)
	}
	entries := strings.Split(strings.TrimSpace(string(data)), "\n")
	if *lines > 0 && len(entries) > *lines {
		entries = entries[len(entries)-*lines:]
	}
	for _, line := range entries {
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			fmt.Println(line)
			continue
		}
		fmt.Printf("%s [%s] %s: %s\n", entry.Timestamp, entry.Level, entry.Service, entry.Message)
	}
}
