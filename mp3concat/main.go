package main

import (
	"fmt"
	"os"
)

type mp3File struct {
	fileName string
}

const (
	allowedFileExt     = ".mp3"
	requiredFiledCount = 3
)

var (
	fileCount int
	files     []mp3File
)

func init() {
	parseArgs()
	if fileCount < requiredFiledCount {
		panic("Not Enoug Files to Concat")
	}
}

func main() {

}

func parseArgs() {
	args := os.Args[1:]
	for _, fileName := range args {
		fileExtension := string(fileName[len(fileName)-4:])
		if fileExtension != allowedFileExt {
			fmt.Println(fileExtension)
			fmt.Println(fileName)
			panic("All files must end with .mp3")
		}
		newFile := mp3File{fileName: fileName}
		files = append(files, newFile)
		fileCount++
	}
}
