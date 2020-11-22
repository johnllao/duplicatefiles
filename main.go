package main

import (
	"flag"
	"log"

	"github.com/johnllao/duplicatefiles/app"
)

var (
	dbpath     string
	searchpath string
)

func init() {
	flag.StringVar(&dbpath,     "d", "", "path of the database file")
	flag.StringVar(&searchpath, "s", "", "path to search for duplicate files")
	flag.Parse()
}

func main() {
	var err error
	var a  = app.NewApp(searchpath, dbpath)
	err = a.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Done")
}
