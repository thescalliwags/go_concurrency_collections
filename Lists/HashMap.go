package Lists

import (
	"container/list"
	"fmt"
	"os"
)

/* HashMap uses the Hash function in Lists/HashFunc
to produce a hash for a given key. This key,val pair
is stored in a linked list of a given type (ListType).
All further ops work on the hash value mod #buckets
*/
type HashMap struct {
	buckets    []List
	numBuckets uint64
}

type ListType int

const (
	CGListType = iota
	LFListType
	LLListType
)

/* Convert string of list type to concrete ListType */
func ParseType(str string) ListType {
	switch str {
	case "CG":
		return CGListType
	case "LF":
		return LFListType
	case "LL":
		return LLListType
	default:
		fmt.Printf("Must supply list type: either CG, LF, or LL\n")
		os.Exit(1)
		return CGListType
	}
}

func (hm *HashMap) Init(numBuckets int, listType ListType) {
	hm.numBuckets = uint64(numBuckets)

	hm.buckets = make([]List, numBuckets)

	for i := 0; i < numBuckets; i++ {
		switch listType {
		case CGListType:
			hm.buckets[i] = new(CGList)
		case LFListType:
			hm.buckets[i] = new(LFList)
		case LLListType:
			hm.buckets[i] = new(LazyList)
		default:
			fmt.Printf("improper hashmap type detected\n")
			os.Exit(1)
		}

		hm.buckets[i].Init()
	}
}

/* Likely not mutli-thread safe. However, will be called after synchronized state */
func (hm *HashMap) KeysAndValues() (*list.List, *list.List) {
	keys := list.New()
	values := list.New()

	for _, bucket := range hm.buckets {
		thesekeys, thesevalues := bucket.KeysAndValues()
		for e := thesekeys.Front(); e != nil; e = e.Next() {
			keys.PushBack(e.Value)
		}
		for e := thesevalues.Front(); e != nil; e = e.Next() {
			values.PushBack(e.Value)
		}
	}
	return keys, values
}

func (hm *HashMap) Get(key interface{}) (interface{}, bool) {
	var keyHash uint64
	hash32, _ := getHash(key)
	keyHash = uint64(hash32)

	bucketId := keyHash % hm.numBuckets

	return hm.buckets[bucketId].Get(key)
}

func (hm *HashMap) Remove(key interface{}) bool {
	var keyHash uint64
	hash32, _ := getHash(key)
	keyHash = uint64(hash32)

	bucketId := keyHash % hm.numBuckets

	return hm.buckets[bucketId].Remove(key)
}

func (hm *HashMap) Insert(key interface{}, val interface{}) bool {
	var keyHash uint64
	hash32, _ := getHash(key)
	keyHash = uint64(hash32)

	bucketId := keyHash % hm.numBuckets

	return hm.buckets[bucketId].Insert(key, val)
}

func (hm *HashMap) PrintMap() {
	for i := 0; i < int(hm.numBuckets); i++ {
		hm.buckets[i].Printlist()
	}
}
