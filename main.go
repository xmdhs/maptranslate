package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/xmdhs/maptranslate/chunk"
	"github.com/xmdhs/maptranslate/model"
)

func main() {
	cxt := context.Background()

	l, err := getForDataDir(cxt, "region")
	if err != nil {
		panic(err)
	}
	newL := make([]chunk.Region[model.NbtHasText], 0)
	for i := range l {
		for ii := range l[i].Chunk {
			l[i].Chunk[ii].RemoveNull()
		}
		l[i].RemoveNull()
		if len(l[i].Chunk) != 0 {
			newL = append(newL, l[i])
		}
	}
	b, err := json.Marshal(newL)
	if err != nil {
		panic(err)
	}
	os.WriteFile("data.json", b, 0700)
}

func getForDataDir(cxt context.Context, dirname string) ([]chunk.Region[model.NbtHasText], error) {
	dir, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("getForDataDir: %w", err)
	}
	cl := make([]chunk.Region[model.NbtHasText], 0)
	clCh := make(chan chunk.Region[model.NbtHasText], 50)
	errCh := make(chan error, 10)

	numcpu := runtime.NumCPU()

	cxt, cancel := context.WithCancel(cxt)
	defer cancel()

	go func() {
		i := 0
		w := sync.WaitGroup{}
		for _, f := range dir {
			f := f
			if f.IsDir() {
				continue
			}
			w.Add(1)
			i++
			go func() {
				defer w.Done()
				path := filepath.Join(dirname, f.Name())
				f, err := os.Open(path)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("getForDataDir: %w", err):
					case <-cxt.Done():
					}
					return
				}
				v, err := chunk.PaseMca[model.NbtHasText](f, path)
				if err != nil {
					select {
					case errCh <- fmt.Errorf("getForDataDir: %w", err):
					case <-cxt.Done():
					}
					return
				}
				select {
				case clCh <- v:
				case <-cxt.Done():
				}

			}()
			if i > numcpu {
				w.Wait()
				i = 0
			}
			select {
			case <-cxt.Done():
				return
			default:
			}
		}
		w.Wait()
		cancel()
	}()

	for {
		select {
		case <-cxt.Done():
			return cl, nil
		case c := <-clCh:
			cl = append(cl, c)
		case err := <-errCh:
			return nil, err
		}
	}
}
