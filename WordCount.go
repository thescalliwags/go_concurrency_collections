package main

import (
	"./Lists"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var numbuckets = 2 * 1024
var numthreads = 32

//Taken from https://stackoverflow.com/questions/5884154/golang-read-text-file-into-string-array-and-write
// Read a whole file into the memory and store it as array of lines
func readLines(path string) (lines []string, err error) {
	var (
		file   *os.File
		part   []byte
		prefix bool
	)
	if file, err = os.Open(path); err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := bytes.NewBuffer(make([]byte, 0))
	for {
		if part, prefix, err = reader.ReadLine(); err != nil {
			break
		}
		buffer.Write(part)
		if !prefix {
			lines = append(lines, buffer.String())
			buffer.Reset()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

/* For use with concurrency so if a thread finishes early,
it can find new work to do */
type section struct {
	startln int
	endln   int //Should stop at this line (not read it)
	done    bool
	lock    *sync.Mutex
}

func initSections(lines []string, numthreads int) []*section {
	var numsections = (int)((float64)(numthreads) * math.Log2((float64)(numthreads)))
	if numsections == 0 { //Can't' have 0 sections
		numsections = 1
	}
	sections := make([]*section, numsections)
	startln := 0
	sectionln := numsections / len(lines)
	//Partition the lines array into numthreads*ln(numthreads) sections
	for i := 0; i < numsections-1; i++ {
		thissection := new(section)
		thissection.startln = startln
		thissection.endln = startln + sectionln
		thissection.done = false
		thissection.lock = &sync.Mutex{}
		startln = startln + sectionln
		sections[i] = thissection
	}
	//Last section, make sure it ends with the last line
	thissection := new(section)
	thissection.startln = startln
	thissection.endln = len(lines)
	thissection.done = false
	thissection.lock = &sync.Mutex{}
	startln = startln + sectionln
	sections[len(sections)-1] = thissection

	return sections
}

func wcConcurrent(lines []string) time.Duration {
	hmap := make(map[string]int)
	start := time.Now()
	for i := 0; i < len(lines); i++ {
		words := strings.Fields(lines[i])
		for _, word := range words {
			//clean up word (trim whitespace and spec chars
			word = strings.ToLower(strings.Trim(word, ".,;*'`\":?!\\ {}()/"))
			val, ok := hmap[word]
			if ok {
				hmap[word] = val + 1
			} else {
				hmap[word] = 1
			}
		}
	}
	elapsed := time.Since(start)
	return elapsed
}

func wcParallel(lines []string, hmap *Lists.HashMap, numthreads int) time.Duration {
	sections := initSections(lines, numthreads)
	var wg sync.WaitGroup
	wg.Add(numthreads)
	start := time.Now()
	for i := 0; i < numthreads; i++ {
		go countlines(hmap, &wg, sections, lines, i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	return elapsed
}

func countlines(hmap *Lists.HashMap, wg *sync.WaitGroup, sections []*section, lines []string, startsection int) {
	validsec := true
	var chosensection *section
	for { //Until no valid sections left
		validsec = false
		//Search for an unstarted section
		for i := startsection; i < len(sections); i++ {
			if sections[i].done == false {
				sections[i].lock.Lock()
				if sections[i].done == false {
					//Found valid section to work on
					sections[i].done = true
					validsec = true
					chosensection = sections[i]
					break
				} else {
					sections[i].lock.Unlock()
					continue
				}
			}
		}
		if validsec == false {
			break
		}
		//Actually do the work
		dosection(hmap, chosensection, lines)
		chosensection.lock.Unlock()
	}
	wg.Done()
}

func dosection(hmap *Lists.HashMap, chosensection *section, lines []string) {
	for i := chosensection.startln; i < chosensection.endln; i++ {
		words := strings.Fields(lines[i])
		for _, word := range words {
			//clean up word (trim whitespace and spec chars
			word = strings.ToLower(strings.Trim(word, ".,;*'`\":?!\\[] {}()/"))
			val, there := hmap.Get(word)
			if there == false {
				zero := new(int32)
				*zero = 0
				val = zero
				hmap.Insert(word, val)
			}
			count := val.(*int32)
			//CAS to increment pointer
			for {
				currentval := *count
				if atomic.CompareAndSwapInt32(count, currentval, currentval+1) {
					break
				}
			}
		}
	}

}

func main() {
	//Command line input
	var listTypeStr = flag.String("file", "texts/OriginOfSpecies.txt", "Which file to perform wc on")
	flag.Parse()

	lines, err := readLines(*listTypeStr)
	if err != nil {
		fmt.Println("Error reading file!")
		os.Exit(1)
	}
	hMap := new(Lists.HashMap)
	hMap.Init(numbuckets, Lists.LLListType)

	/* Parallel compute */
	paralleltime := wcParallel(lines, hMap, numthreads)
	/* Concurrent compute */
	concurrenttime := wcConcurrent(lines)

	//Report results
	keys, values := hMap.KeysAndValues()
	maxkey := ""
	maxval := (int32)(0)
	//Output map
	v := values.Front()
	for k := keys.Front(); k != nil; k = k.Next() {
		s := k.Value.(string)
		n := v.Value.(*int32)
		if *n > maxval {
			maxval = *n
			maxkey = s
		}
		fmt.Printf("%s -> %d\n", s, *n)
		v = v.Next()
	}
	//Report info
	fmt.Printf("Most often used word is '%s', used %d times\n", maxkey, maxval)
	fmt.Printf("Parallel took: %s\nConcurrent took: %s\n", paralleltime, concurrenttime)
}