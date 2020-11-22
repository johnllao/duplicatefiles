package app

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
)

type App struct {
	db         *bolt.DB
	dbpath     string
	searchpath string
}

func NewApp(s, d string) *App {
	return &App {
		dbpath:     d,
		searchpath: s,
	}
}

func (a *App) Start() error {
	var err error

	var uniqueid string
	uniqueid, err = uid()
	if err != nil {
		return err
	}

	var dbfilepath = filepath.Join(a.dbpath, uniqueid + ".db")

	a.db, err = bolt.Open(dbfilepath, 0600, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = a.db.Close()
		_ = os.Remove(dbfilepath)
	}()

	log.Printf("searching for duplicate files. path: %s", a.searchpath)

	var dinfos []os.FileInfo
	dinfos, err = ioutil.ReadDir(a.searchpath)
	if err != nil {
		return err
	}

	for i := 0; i < len(dinfos); i++ {
		log.Printf("%s - %v", dinfos[i].Name(), dinfos[i].IsDir())
	}

	return err
}

