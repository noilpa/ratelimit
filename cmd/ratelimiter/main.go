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
	rate := flag.Uint("rate", 1, "method's rate limit")
	inflight := flag.Int("inflight", 1, "simultaneously methods inflight")
	withTotalTime := flag.Bool("time", false, "print total execution time")
	flag.Parse()
	args := flag.Args()

	if *inflight <= 0 {
		panic("invalid inflight number: value must be greater than 0")
	}

	if len(args) < 2 {
		panic("invalid target method")
	}

	m := args[0]
	opts := strings.Join(args[1:], " ")

	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	stdinArgs := strings.Split(string(stdin), "\n")
	jobs := make(chan string, len(stdinArgs))
	go func() {
		for _, arg := range stdinArgs {
			jobs <- arg
		}
		close(jobs)
	}()

	timeout := time.Second / time.Duration(*rate)

	f := func(arg string) {
		cmd := exec.Command(m, strings.Split(strings.ReplaceAll(opts, "{}", arg), " ")...)
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("%s, err: %v\n", cmd.String(), err)
			return
		}
		fmt.Print(string(output))
	}

	newPool(*inflight).do(jobs, f, timeout)

	if *withTotalTime {
		fmt.Println("##########")
		fmt.Println("total time: ", time.Since(start).String())
	}
}

func newPool(size int) *pool {
	return &pool{
		size: size,
		wg:   new(sync.WaitGroup),
	}
}

type pool struct {
	size int
	wg   *sync.WaitGroup
}

func (p pool) do(jobs <-chan string, f func(arg string), timeout time.Duration) {
	p.wg.Add(p.size)
	for i := 0; i < p.size; i++ {
		w := &worker{
			timeout: timeout,
		}
		go w.run(jobs, f, p.wg)
	}
	p.wg.Wait()
}

type worker struct {
	lastTaskStart time.Time
	timeout       time.Duration
}

func (w worker) await() {
	delta := time.Now().Sub(w.lastTaskStart.Add(w.timeout))
	if delta < 0 {
		time.Sleep(-delta)
	}
	return
}

func (w *worker) run(jobs <-chan string, f func(arg string), wg *sync.WaitGroup) {
	for {
		select {
		case j, ok := <-jobs:
			if ok {
				w.await()
				w.lastTaskStart = time.Now()
				f(j)
			} else {
				wg.Done()
				return
			}
		}
	}
}
