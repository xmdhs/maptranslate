package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/xmdhs/maptranslate/chunk"
	"github.com/xmdhs/maptranslate/model"
)

func main() {
	bs := bufio.NewScanner(strings.NewReader("4\n"))
	//bs := bufio.NewScanner(os.Stdin)

	fmt.Println("你想要：")
	fmt.Println("1. 读取方块实体和实体的 nbt 信息")
	fmt.Println("2. 应用 json 文件到 nbt 中")
	fmt.Println("3. data.json 转换成 lang.json （相同文本合并）")
	fmt.Println("4. lang.json 转 data.json")
	fmt.Print("> ")
	bs.Scan()

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			bs.Scan()
		}
	}()

	switch bs.Text() {
	case "1":
		cxt := context.Background()
		l, err := getForDataDir(cxt, `region`)
		if err != nil {
			fmt.Println(`region`, err)
		}
		ll, err := getForDataDir(cxt, `entities`)
		if err != nil {
			fmt.Println(`entities`, err)
		}
		l = append(l, ll...)

		newL := []chunk.Region[[]chunk.Entities]{}
		for _, v := range l {
			v.RemoveNull()
			if len(v.Chunk) != 0 {
				newL = append(newL, v)
			}
		}
		wirteJson("data.json", newL)
		fmt.Println("完成，已写入 data.json")
	case "2":
		list := readDatajson[[]chunk.Region[[]chunk.Entities]]("data.json")
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
				w.Done()
			}()
			if i > numcpu {
				w.Wait()
				i = 0
			}
		}
		w.Wait()
		fmt.Println("完成")
	case "3":
		l := regionList2LangList(readDatajson[[]chunk.Region[[]chunk.Entities]]("data.json"))
		wirteJson("lang.json", l)

	case "4":
		list := readDatajson[[]chunk.Region[[]chunk.Entities]]("data.json")
		langL := readDatajson[[]Lang]("lang.json")
		useLangList(langL, &list)
		wirteJson("data.json", list)
	}
	bs.Scan()
}

func readDatajson[K any](path string) K {
	bb, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var v K
	err = json.Unmarshal(bb, &v)
	if err != nil {
		panic(err)
	}
	return v
}

func getForDataDir(cxt context.Context, dirname string) ([]chunk.Region[[]chunk.Entities], error) {
	dir, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("getForDataDir: %w", err)
	}
	cl := make([]chunk.Region[[]chunk.Entities], 0)
	clCh := make(chan chunk.Region[[]chunk.Entities], 50)
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
			info, err := f.Info()
			if err != nil {
				select {
				case errCh <- fmt.Errorf("getForDataDir: %w", err):
				case <-cxt.Done():
				}
				return
			}
			if info.Size() == 0 {
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

func wirteJson(filename string, v any) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	en := json.NewEncoder(f)
	en.SetEscapeHTML(false)
	en.SetIndent("", "    ")
	err = en.Encode(v)
	if err != nil {
		panic(err)
	}
}
