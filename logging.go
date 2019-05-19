package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func logPath() string {
	year, month, day := time.Now().Date()
	return fmt.Sprintf("logs/%d-%d-%d.log", year, month, day)
}

func openLog(path string) (*os.File, error) {
	if _, err := os.Open(path); os.IsNotExist(err) {
		_, err = os.Create(path)
		if err != nil {
			panic(err)
		}
	}

	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModeAppend|0644)
}

// StartLog sets up logging
func StartLog() *os.File {
	if _, err := os.Open("logs"); os.IsNotExist(err) {
		os.Mkdir("logs", 0700)
	}
	logFile, err := openLog(logPath())
	if err != nil {
		panic(err)
	}
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.Ltime + log.Lshortfile)
	return logFile
}

// CompressLog replaces a log with a gzip-compressed log
func CompressLog(logFile *os.File) {
	err := logFile.Close()
	if err != nil {
		panic(err)
	}
	f, err := openLog(logPath() + ".gz")
	if err != nil {
		panic(err)
	}
	l, err := os.Open(logPath())
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(l)
	if err != nil {
		panic(err)
	}
	l.Close()
	gz := gzip.NewWriter(f)
	gz.Write(b)
	err = gz.Close()
	if err != nil {
		panic(err)
	}
	err = os.Remove(logPath())
	if err != nil {
		panic(err)
	}

}
