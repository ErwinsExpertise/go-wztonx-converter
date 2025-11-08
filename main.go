// NoLifeWzToNx - Go implementation
// Converts WZ files into NX files
// Based on https://github.com/NoLifeDev/NoLifeStory/blob/master/src/wztonx/wztonx.cpp

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// Version information (set by GoReleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Println("NoLifeWzToNx - Go Edition")
	fmt.Printf("Version: %s (commit: %s, built: %s)\n", version, commit, date)
	fmt.Println("Converts WZ files into NX files")
	fmt.Println()

	client := flag.Bool("client", false, "Client mode (process audio and bitmaps)")
	clientShort := flag.Bool("c", false, "Client mode (short)")
	server := flag.Bool("server", false, "Server mode")
	serverShort := flag.Bool("s", false, "Server mode (short)")
	lz4hc := flag.Bool("lz4hc", false, "Use LZ4 high compression")
	lz4hcShort := flag.Bool("h", false, "Use LZ4 high compression (short)")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := flag.String("memprofile", "", "Write memory profile to file")
	flag.Parse()

	// CPU profiling
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal("Could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("Could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("CPU profiling enabled, writing to %s\n", *cpuProfile)
	}

	// Memory profiling (defer to end of main)
	if *memProfile != "" {
		defer func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				log.Fatal("Could not create memory profile: ", err)
			}
			defer f.Close()
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("Could not write memory profile: ", err)
			}
			fmt.Printf("Memory profile written to %s\n", *memProfile)
		}()
	}

	isClient := *client || *clientShort
	isServer := *server || *serverShort
	useHC := *lz4hc || *lz4hcShort

	// If server is specified, client is false
	if isServer {
		isClient = false
	}

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Println("Usage: go-wztonx-converter [options] <files/directories>")
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	startTime := time.Now()

	for _, path := range paths {
		if err := processPath(path, isClient, useHC); err != nil {
			log.Printf("Error processing %s: %v\n", path, err)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Took %d seconds\n", int(elapsed.Seconds()))
}

func processPath(path string, client bool, hc bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				return convertFile(p, client, hc)
			}
			return nil
		})
	}

	return convertFile(path, client, hc)
}

func convertFile(filename string, client bool, hc bool) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".wz" && ext != ".img" {
		return nil
	}

	nxFilename := strings.TrimSuffix(filename, ext) + ".nx"
	fmt.Printf("%s -> %s\n", filename, nxFilename)

	converter := NewConverter(filename, nxFilename, client, hc)
	return converter.Convert()
}
