package main

import (
	"bufio"
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
	bs := bufio.NewScanner(os.Stdin)

	fmt.Println("你想要：")
	fmt.Println("1. 读取方块实体和实体的 nbt 信息")
	fmt.Println("2. 应用 json 文件到 nbt 中")
	bs.Scan()
	switch bs.Text() {
	case "1":
		cxt := context.Background()
		l, err := getForDataDir(cxt, `region`)
		if err != nil {
			panic(err)
		}
		ll, err := getForDataDir(cxt, `entities`)
		if err != nil {
			panic(err)
		}
		l = append(l, ll...)
		newL := make([]chunk.Region[model.NbtHasText], 0)
		for i := range l {
			for ii := range l[i].Chunk {
				chunk.ChunkRemoveNullSlice(&l[i].Chunk[ii].Data)
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
		os.WriteFile("data1.json", b, 0700)
	case "2":
		bb, err := os.ReadFile("data.json")
		if err != nil {
			panic(err)
		}
		list := []chunk.Region[model.NbtHasText]{}
		err = json.Unmarshal(bb, &list)
		if err != nil {
			panic(err)
		}
		i := 0
		w := sync.WaitGroup{}
		numcpu := runtime.NumCPU()

		for _, v := range list {
			i++
			w.Add(1)
			v := v
			go func() {
				err := chunk.WriteChunk(v)
				if err != nil {
					panic(err)
				}
			}()
			if i > numcpu {
				w.Wait()
				i = 0
			}
		}

	}
	bs.Scan()
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
			name := f.Name()
			if f.IsDir() || filepath.Ext(name) != ".mca" {
				continue
			}
			w.Add(1)
			i++
			go func() {
				defer w.Done()
				path := filepath.Join(dirname, name)
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
