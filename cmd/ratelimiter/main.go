package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func main() {
	start := time.Now()
	flag.Args()
	method := flag.String("m", "echo", "target method")
	rate := flag.Uint("rate", 1, "rate limit")
	inflight := flag.Int("inflight", 1, "simultaneously inflight rate limiters")
	withTime := flag.Bool("time", false, "print execution time")
	flag.Parse()

	if *inflight <= 0 {
		panic("invalid inflight number: value must be greater than 0")
	}

	splitted := strings.Split(*method, " ")
	m := splitted[0]
	opts := strings.Join(splitted[1:], " ")

	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	args := strings.Split(string(stdin), "\n")
	jobs := make(chan string, len(args))
	for _, arg := range args {
		jobs <- arg
	}
	close(jobs)

	timeout := time.Second / time.Duration(*rate)
	fmt.Printf("timeout: %d\n", timeout)

	f := func(arg string) {
		cmd := exec.Command(m, strings.TrimSpace(strings.Join([]string{opts, arg}, " ")))
		fmt.Println()
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("%s, err: %v\n", cmd.String(), err)
			return
		}
		fmt.Printf("%s -> %s\n", cmd.String(), output)
	}

	wg := new(sync.WaitGroup)
	wg.Add(*inflight)
	for i := 0; i < *inflight; i++ {
		w := &worker{
			id:      i,
			timeout: timeout,
			wg:      wg,
		}
		go w.run(jobs, f)
	}
	wg.Wait()

	if *withTime {
		fmt.Println("##########")
		fmt.Println("total time: ", time.Since(start).String())
	}
}

type worker struct {
	id            int
	lastTaskStart time.Time
	timeout       time.Duration
	wg            *sync.WaitGroup
}

func (w worker) await() {
	delta := time.Now().Sub(w.lastTaskStart.Add(w.timeout))
	fmt.Println("worker ", w.id, " await ", delta.String())
	if delta < 0 {
		time.Sleep(-delta)
	}
	return
}

func (w *worker) run(jobs <-chan string, f func(arg string)) {
	fmt.Println("worker ", w.id, " run")
	for {
		select {
		case j, ok := <-jobs:
			if ok {
				w.await()
				w.lastTaskStart = time.Now()
				fmt.Println("worker ", w.id, " start ", j)
				f(j)
				fmt.Println("worker ", w.id, " done ", j)
			} else {
				w.wg.Done()
				fmt.Println("worker ", w.id, " gone")
				return
			}
		}
	}
}
