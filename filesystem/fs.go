//Package filesystem provides a filesystem cache implementation for onecache
package filesystem

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/adelowo/onecache"
)

const (
	defaultFilePerm          os.FileMode = 0666
	defaultDirectoryFilePerm             = 0755
)

func createDirectory(dir string) error {

	return os.MkdirAll(dir, defaultDirectoryFilePerm)
}

type FSStore struct {
	baseDir string
}

//Returns an initialized Filesystem Cache
//If a non-existent directory is passed, it would be created automatically.
//This function Panics if the directory could not be created
func MustNewFSStore(baseDir string) *FSStore {

	_, err := os.Stat(baseDir)

	if err != nil { //Directory does not exist..Let's create it
		if err := createDirectory(baseDir); err != nil {
			panic(fmt.Errorf("Base directory could not be created : %s", err))
		}
	}

	return &FSStore{baseDir}
}

func (fs *FSStore) Set(key string, data interface{}, expiresAt time.Duration) error {

	path := fs.filePathFor(key)

	if err := os.MkdirAll(filepath.Dir(path), defaultDirectoryFilePerm); err != nil {
		return err
	}

	i := &onecache.Item{ExpiresAt: time.Now().Add(expiresAt), Data: data}

	b, err := i.Bytes()

	if err != nil {
		return err
	}

	return writeFile(path, b)
}

//Fetches a cache key.
//This runs garbage collection on the key if necessary
func (fs *FSStore) Get(key string) (interface{}, error) {

	b, err := ioutil.ReadFile(fs.filePathFor(key))

	if err != nil {
		return nil, err
	}

	i, err := onecache.BytesToItem(b)

	if i.IsExpired() {
		fs.Delete(key)
		return nil, onecache.ErrCacheMiss
	}

	return i.Data, nil
}

//Removes a file (cached item) from the disk
func (fs *FSStore) Delete(key string) error {
	return os.RemoveAll(fs.filePathFor(key))
}

//Cleans up the entire cache
func (fs *FSStore) Flush() error {
	return os.RemoveAll(fs.baseDir)
}

//Gets a unique path for a cache key.
//This is going to be a directory 3 level deep. Something like "basedir/33/rr/33/hash"
func (fs *FSStore) filePathFor(key string) string {
	hashSum := md5.Sum([]byte(key))

	hashSumAsString := hex.EncodeToString(hashSum[:])

	return filepath.Join(fs.baseDir,
		string(hashSumAsString[0:2]),
		string(hashSumAsString[2:4]),
		string(hashSumAsString[4:6]), hashSumAsString)
}

func (fs *FSStore) Increment(key string, steps int) error {

	path := fs.filePathFor(key)

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	item, err := onecache.BytesToItem(b)

	if err != nil {
		return err
	}

	item.Data, err = onecache.Increment(item.Data, steps)

	if err != nil {
		return err
	}

	b, err = item.Bytes()

	if err != nil {
		return err
	}

	return writeFile(path, b)
}

func (fs *FSStore) Decrement(key string, steps int) error {

	path := fs.filePathFor(key)

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	item, err := onecache.BytesToItem(b)

	if err != nil {
		return err
	}

	item.Data, err = onecache.Decrement(item.Data, steps)

	if err != nil {
		return err
	}

	b, err = item.Bytes()

	if err != nil {
		return err
	}

	return writeFile(path, b)
}

func writeFile(path string, b []byte) error {
	return ioutil.WriteFile(path, b, defaultFilePerm)
}