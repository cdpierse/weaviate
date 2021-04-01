package lsmkv

import (
	"bytes"
	"encoding/binary"
	"os"
	"sort"
	"sync"

	"github.com/pkg/errors"
	"github.com/spaolacci/murmur3"
)

type Memtable struct {
	sync.RWMutex
	key          *binarySearchTree
	primaryIndex *binarySearchTree
	size         uint64
	path         string
}

func newMemtable(path string) *Memtable {
	return &Memtable{
		key:          &binarySearchTree{},
		primaryIndex: &binarySearchTree{}, // todo, sort upfront
		path:         path,
	}
}

type keyIndex struct {
	hash       []byte
	valueStart int
	valueEnd   int
}

func (l *Memtable) get(key []byte) ([]byte, error) {
	l.RLock()
	defer l.RUnlock()

	v, ok := l.key.get(key)
	if !ok {
		return nil, NotFound
	}

	return v, nil
}

func (l *Memtable) put(key, value []byte) error {
	l.Lock()
	defer l.Unlock()
	l.key.insert(key, value)
	l.size += uint64(len(key))
	l.size += uint64(len(value))

	return nil
}

func (l *Memtable) Size() uint64 {
	l.RLock()
	defer l.RUnlock()

	return l.size
}

func (l *Memtable) flush() error {
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}

	defer f.Close()

	flat := l.key.flattenInOrder()
	indexPos := uint64(totalValueSize(flat)) + 10 // for the level and indicator offset itself
	level := uint16(0)                            // always level zero on a new one

	if err := binary.Write(f, binary.LittleEndian, &level); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, &indexPos); err != nil {
		return err
	}

	keys := make([]keyIndex, len(flat))

	totalWritten := 10 // offset level + indexPos unit64
	for i, node := range flat {
		n, err := f.Write(node.value)
		if err != nil {
			return errors.Wrapf(err, "write node %d", i)
		}

		hasher := murmur3.New128()
		hasher.Write(node.key)
		hash := hasher.Sum(nil)
		keys[i] = keyIndex{
			valueStart: totalWritten,
			valueEnd:   totalWritten + n,
			hash:       hash,
		}

		totalWritten += n
	}

	// now sort keys according to their hashes for an efficient binary search
	sort.Slice(keys, func(a, b int) bool {
		return bytes.Compare(keys[a].hash, keys[b].hash) < 0
	})

	// now write all the keys with "links" to the values
	// delimit a key with \xFF (obviously needs a better mechanism to protect against the data containing the delimter byte)
	for _, key := range keys {
		f.Write(key.hash)

		start := uint64(key.valueStart)
		end := uint64(key.valueEnd)
		if err := binary.Write(f, binary.LittleEndian, &start); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, &end); err != nil {
			return err
		}
	}

	return nil
}

func totalValueSize(in []*binarySearchNode) int {
	var sum int
	for _, n := range in {
		sum += len(n.value)
	}

	return sum
}
