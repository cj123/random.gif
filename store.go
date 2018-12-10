package main

import (
	"net/http"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"encoding/hex"
	"crypto/md5"
	"sync"
	"encoding/json"
	"errors"
	"strings"
)

var (
	gifAlreadyExistsError = errors.New("gif present in index")
	gifNotInIndexError = errors.New("gif not in index")
)

type gifStore interface {
	Init() error
	Store([]byte, gifURL) (string, error)
	Get(string) ([]byte, error)
	Delete(gifURL) error
	All() map[string]*gif
}

type gif struct {
	Location string `json:"loc"`
	URL      gifURL `json:"url"`
}

type diskStore struct {
	loc   string
	index map[string]*gif
	mutex *sync.Mutex
}

func newDiskStore(dest string) *diskStore {
	return &diskStore{
		index:  make(map[string]*gif),
		mutex: &sync.Mutex{},
		loc: dest,
	}
}

func (d *diskStore) Init() error {
	indexPath := path.Join(d.loc, "index.json")

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		err := d.writeIndex()

		if err != nil {
			return err
		}
	}

	return d.readIndex()
}

func (d *diskStore) appendToIndex(url gifURL, dir string) error {
	d.index[d.makeIdentifier(url)] = &gif{
		Location: dir,
		URL: url,
	}

	return d.writeIndex()
}

func (d *diskStore) writeIndex() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	saveFile := path.Join(d.loc, "index.json")

	b, err := json.Marshal(d.index)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(saveFile, b, 0644)
}

func (d *diskStore) readIndex() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	saveFile := path.Join(d.loc, "index.json")

	b, err := ioutil.ReadFile(saveFile)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, &d.index)
}

func (d *diskStore) Store(b []byte, gif gifURL) (string, error) {
	id := d.makeIdentifier(gif)

	if _, ok := d.index[d.makeIdentifier(gif)]; ok {
		return id, gifAlreadyExistsError
	}

	u, err := url.Parse(string(gif))

	if err != nil {
		return "", err
	}

	saveDir := path.Join(d.loc, u.Host, u.Path)

	err = os.MkdirAll(path.Dir(saveDir), os.ModePerm)

	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(saveDir, b, 0644)

	if err != nil {
		return "", err
	}

	// update index without save location
	return id, d.appendToIndex(gif, path.Join(u.Host, u.Path))
}

func (d *diskStore) Get(hash string) ([]byte, error) {
	gif, ok := d.index[hash]

	if !ok {
		return nil, gifNotInIndexError
	}

	return ioutil.ReadFile(path.Join(d.loc, gif.Location))
}

func (d *diskStore) Delete(gif gifURL) error {
	for key, g := range d.index {
		if g.URL == gif {
			delete(d.index, key)
		}
	}

	return d.writeIndex()
}

func (d *diskStore) All() map[string]*gif {
	return d.index
}

func download(url gifURL) ([]byte, error) {
	u := string(url)

	if strings.Contains(u, ".gifv") {
		// assume this is imgur and we can have .gif
		u = strings.Replace(u, ".gifv", ".gif", -1)
	}

	resp, err := http.Get(u)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (d *diskStore) makeIdentifier(gif gifURL) string {
	hasher := md5.New()
	hasher.Write([]byte(gif))
	return hex.EncodeToString(hasher.Sum(nil))
}
