package app

import (
	"crypto/sha256"
	"encoding/binary"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/boltdb/bolt"
)

type App struct {
	db         *bolt.DB
	dbpath     string
	searchpath string
	dups       [][]string

	opendb     func(string, os.FileMode, *bolt.Options) (*bolt.DB, error)
	readdir    func(string) ([]os.FileInfo, error)
	readfile   func(string) ([]byte, error)
}

func NewApp(s, d string) *App {
	return &App {
		dbpath:     d,
		searchpath: s,

		opendb:     bolt.Open,
		readdir:    ioutil.ReadDir,
		readfile:   ioutil.ReadFile,
	}
}

func (a *App) Start() error {
	var err error

	// unique id for the boltdb data file
	var uniqueid string
	uniqueid, err = uid()
	if err != nil {
		return err
	}

	// create bolddb file
	var dbfilepath = filepath.Join(a.dbpath, uniqueid + ".db")
	a.db, err = a.opendb(dbfilepath, 0600, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = a.db.Close()
		_ = os.Remove(dbfilepath)
	}()

	var quitc = make(chan os.Signal)
	signal.Notify(quitc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quitc
		_ = a.db.Close()
		_ = os.Remove(dbfilepath)
		os.Exit(1)
	}()

	log.Printf("searching for duplicate files. path: %s", a.searchpath)

	// map to track duplicates
	var m = make(map[[sha256.Size]byte]int)

	// queue for the path to search
	// we start from the path provided in the argument
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
				filedata, err = a.readfile(filepath)
				if err != nil {
					return err
				}

				var h = sha256.New()
				_, err = h.Write(filedata)
				if err != nil {
					return err
				}

				// convert the hash slice to a fixed array
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

	a.dups = make([][]string, 0)
	for k, v := range m {
		if v > 1 {
			a.db.View(func(tx *bolt.Tx) error {
				var b = tx.Bucket(k[:])
				var cur = b.Cursor()

				var dupfiles = make([]string, 0)
				for kk, vv := cur.First(); kk != nil; kk, vv = cur.Next() {
					dupfiles = append(dupfiles, string(vv))
				}
				a.dups = append(a.dups, dupfiles)
				return nil
			})
		}
	}

	log.Printf("found %d files has duplicates", len(a.dups))

	return a.HTTP()
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

func (a *App) HTTP() error {
	var s = http.Server {
		Addr: "localhost:8080",
		Handler: http.HandlerFunc(a.roothandler),
	}

	log.Printf("starting service at http://localhost:8080")
	return s.ListenAndServe()
}

func (a *App) roothandler(w http.ResponseWriter, r *http.Request) {

	var err error
	var t *template.Template
	t, err = template.New("DuplicateFiles").Parse(html)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	t.Execute(w, a.dups)
}

var html = `<!DOCTYPE html>
<html>
<head>
	<title>Duplicate Files</title>
	<style type="text/css">
	body {
		background-color: #555;
		color: #91c3dc;
		font-family: Tahoma, Arial;
		font-size: 12pt;
	}
	.group {
		border: 1px solid #aab6a2;
		margin: 1em;
		padding: 1em;
	}
	.list {
		list-style-type: none;
		margin: 0;
		padding: 0;
	}
	.list li {
		padding: .3em;
	}
	</style>
</head>
<body>
	{{ range . }}
	
	<div class="group">
		<ul class="list">
		{{ range . }}
			<li>{{ . }}</li>
    	{{ end }}
		</ul>
	</div>

    {{ end }}
</body>
</html>
`