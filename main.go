package main

import (
	"log"
	"os"

	"BittorrentClient/fileParser"
)

func main() {
	inPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := fileParser.ParseFile(inPath)
	if err != nil {
		log.Fatal(err)
	}

	err = tf.DownloadFile(outPath)
	if err != nil {
		log.Fatal(err)
	}
}
