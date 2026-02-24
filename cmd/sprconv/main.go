// sprconv converts spr_action.sql (MySQL INSERT dump) to spr_action.yaml.
//
// Usage:
//
//	go run ./cmd/sprconv/ [input.sql] [output.yaml]
//
// Defaults: data/sql/spr_action.sql → data/yaml/spr_action.yaml
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type entry struct {
	SprID      int
	ActID      int
	FrameCount int
	FrameRate  int
}

func main() {
	inPath := "data/sql/spr_action.sql"
	outPath := "data/yaml/spr_action.yaml"
	if len(os.Args) >= 2 {
		inPath = os.Args[1]
	}
	if len(os.Args) >= 3 {
		outPath = os.Args[2]
	}

	entries, err := parseSql(inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := writeYaml(outPath, entries); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("converted %d entries → %s\n", len(entries), outPath)
}

func parseSql(path string) ([]entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var entries []entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(strings.ToUpper(line), "INSERT") {
			continue
		}
		start := strings.Index(line, "(")
		end := strings.LastIndex(line, ")")
		if start < 0 || end <= start {
			continue
		}
		fields := strings.Split(line[start+1:end], ",")
		if len(fields) != 4 {
			continue
		}
		sprID, e1 := parseVal(fields[0])
		actID, e2 := parseVal(fields[1])
		fc, e3 := parseVal(fields[2])
		fr, e4 := parseVal(fields[3])
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || fr <= 0 {
			continue
		}
		entries = append(entries, entry{sprID, actID, fc, fr})
	}
	return entries, scanner.Err()
}

func parseVal(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "'\"` ")
	return strconv.Atoi(s)
}

func writeYaml(path string, entries []entry) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "spr_actions:")
	for _, e := range entries {
		fmt.Fprintf(w, "  - spr_id: %d\n", e.SprID)
		fmt.Fprintf(w, "    act_id: %d\n", e.ActID)
		fmt.Fprintf(w, "    framecount: %d\n", e.FrameCount)
		fmt.Fprintf(w, "    framerate: %d\n", e.FrameRate)
	}
	return w.Flush()
}
