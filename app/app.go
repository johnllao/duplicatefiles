package app

import (
	"crypto/sha256"
	"encoding/binary"
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

	opendb     func(string, os.FileMode, *bolt.Options) (*bolt.DB, error)
	readdir    func(string) ([]os.FileInfo, error)
}

func NewApp(s, d string) *App {
	return &App {
		dbpath:     d,
		searchpath: s,

		opendb:     bolt.Open,
		readdir:    ioutil.ReadDir,
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

	a.db, err = a.opendb(dbfilepath, 0600, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = a.db.Close()
		_ = os.Remove(dbfilepath)
	}()

	log.Printf("searching for duplicate files. path: %s", a.searchpath)

	var m = make(map[[sha256.Size]byte]int)

	var searchpaths = make([]string, 1)
	searchpaths[0] = a.searchpath

	for len(searchpaths) > 0 {
		var s = searchpaths[0]
		searchpaths = searchpaths[1:]

		var dinfos []os.FileInfo
		dinfos, err = a.readdir(s)
		if err != nil {
			return err
		}

		for i := 0; i < len(dinfos); i++ {
			if dinfos[i].IsDir() {
				searchpaths = append(searchpaths, filepath.Join(s, dinfos[i].Name()))
			} else {
				var filepath = filepath.Join(s, dinfos[i].Name())
				var filedata []byte
				filedata, err = ioutil.ReadFile(filepath)
				if err != nil {
					return err
				}

				var h = sha256.New()
				_, err = h.Write(filedata)
				if err != nil {
					return err
				}

				var cksum [sha256.Size]byte
				copy(cksum[:], h.Sum(nil))

				a.Save(cksum, filepath)

				var ok bool
				if _, ok = m[cksum]; !ok {
					m[cksum] = 1
				} else {
					m[cksum]++
				}

			}
		}
	}

	for k, v := range m {
		if v > 1 {
			a.db.View(func(tx *bolt.Tx) error {
				var b = tx.Bucket(k[:])
				var cur = b.Cursor()

				for kk, vv := cur.First(); kk != nil; kk, vv = cur.Next() {
					log.Printf("%s", vv)
				}
				return nil
			})
		}
	}

	return err
}

func (a *App) Save(key [sha256.Size]byte, value string) error {
	var err error
	err = a.db.Update(func(tx *bolt.Tx) error {
		var txerr error

		var b *bolt.Bucket
		b, txerr = tx.CreateBucketIfNotExists(key[:])
		if txerr != nil {
			return txerr
		}

		var id uint64
		id, txerr = b.NextSequence()
		if txerr != nil {
			return txerr
		}

		var buf = make([]byte, binary.MaxVarintLen64)
		_ = binary.PutUvarint(buf, id)

		txerr = b.Put(buf, []byte(value))
		if txerr != nil {
			return txerr
		}

		return nil
	})
	return err
}
