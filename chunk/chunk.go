package chunk

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func PaseMca[K any](f io.ReadWriteSeeker, filePath string) (Region[map[string]string], error) {
	rg, err := region.Load(f)
	if err != nil {
		return Region[map[string]string]{}, fmt.Errorf("PaseMca: %w", err)
	}
	defer rg.Close()
	cl := make([]Chunk[map[string]string], 0)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			if !rg.ExistSector(x, y) {
				continue
			}
			b, err := rg.ReadSector(x, y)
			if err != nil {
				return Region[map[string]string]{}, fmt.Errorf("PaseMca: %w", err)
			}
			b, err = mcDecompress(b)
			if err != nil {
				return Region[map[string]string]{}, fmt.Errorf("PaseMca: %w", err)
			}
			var v K
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return Region[map[string]string]{}, fmt.Errorf("PaseMca: %w", err)
			}
			cl = append(cl, Chunk[map[string]string]{
				X:    x,
				Z:    y,
				Data: getStrPath(v, ""),
			})
		}
	}
	return Region[map[string]string]{
		FilePath: filePath,
		Chunk:    cl,
	}, nil
}

var ErrFormt = errors.New("错误的格式")

func BlockPos2Mca(x, z int) string {
	x = int(math.Floor(float64(x) / 512.0))
	z = int(math.Floor(float64(z) / 512.0))
	return fmt.Sprintf("r.%v.%v.mca", x, z)
}

func mcDecompress(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("mcDecompress: %w", ErrInvalidChunk)
	}
	var r io.Reader = bytes.NewReader(data[1:])
	var err error
	switch data[0] {
	default:
		err = fmt.Errorf("testColumn: %w", ErrUnKnownCompression)
	case 1:
		r, err = gzip.NewReader(r)
	case 2:
		r, err = zlib.NewReader(r)
	}
	if err != nil {
		return nil, fmt.Errorf("mcDecompress: %w", err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("mcDecompress: %w", err)
	}
	return b, nil
}

func mcEncompress(data []byte) ([]byte, error) {
	bf := &bytes.Buffer{}
	bf.WriteByte(2)
	zw := zlib.NewWriter(bf)
	defer zw.Close()
	_, err := zw.Write(data)
	if err != nil {
		return nil, fmt.Errorf("mcEncompress: %w", err)
	}
	err = zw.Close()
	if err != nil {
		return nil, fmt.Errorf("mcEncompress: %w", err)
	}
	return bf.Bytes(), nil
}

var (
	ErrInvalidChunk       = errors.New("invalid chunk")
	ErrUnKnownCompression = errors.New("unknown compression")
)

type Region[K any] struct {
	FilePath string
	Chunk    []Chunk[K] `json:",omitempty"`
}

func (r *Region[K]) RemoveNull() {
	newChunk := []Chunk[K]{}
	for _, v := range r.Chunk {
		vv := reflect.ValueOf(v.Data)
		if !vv.IsZero() && vv.Len() != 0 {
			newChunk = append(newChunk, v)
		}
	}
	r.Chunk = newChunk
}

type Chunk[K any] struct {
	X    int
	Z    int
	Data K
}

func getStrPath(v any, path string) map[string]string {
	dt := reflect.TypeOf(v)
	dv := reflect.ValueOf(v)
	if dv.Kind() != reflect.Struct {
		panic("nor struct")
	}
	l := reflect.VisibleFields(dt)
	sm := make(map[string]string)

	for _, t := range l {
		v := dv.FieldByIndex(t.Index)

		nPath := ""
		if path == "" {
			nPath = t.Name
		} else {
			nPath = path + "." + t.Name
		}
		if v.Kind() == reflect.Struct {
			m := getStrPath(v.Interface(), t.Name)
			for k, v := range m {
				sm[k] = v
			}
			continue
		}

		if v.Kind() != reflect.Interface || v.IsNil() {
			continue
		}
		v = v.Elem()
		m, ok := mapify(v.Interface())
		if ok {
			getStrPathMap(m, nPath, &sm)
		}
		if v.Kind() == reflect.Slice {
			getStrPathSlice(v.Interface(), nPath, &sm)
		}
	}
	return sm
}

func getStrPathMap(m map[string]any, path string, sm *map[string]string) {
	for k, v := range m {
		rt := reflect.TypeOf(v)
		rk := rt.Kind()
		switch rk {
		case reflect.Slice:
			getStrPathSlice(v, path+"."+k, sm)
		case reflect.Map:
			getStrPathMap(v.(map[string]any), path+"."+k, sm)
		case reflect.String:
			(*sm)[path+"."+k] = v.(string)
		}
	}
}

func getStrPathSlice(l any, path string, sm *map[string]string) {
	rl := reflect.ValueOf(l)
	rlen := rl.Len()

	for i := 0; i < rlen; i++ {
		ri := rl.Index(i)
		if ri.Kind() == reflect.Interface {
			ri = ri.Elem()
		}
		switch ri.Kind() {
		case reflect.Slice:
			getStrPathSlice(ri.Interface(), path+"["+strconv.Itoa(i)+"]", sm)
		case reflect.Map:
			getStrPathMap(ri.Interface().(map[string]any), path+"["+strconv.Itoa(i)+"]", sm)
		case reflect.String:
			(*sm)[path+"["+strconv.Itoa(i)+"]"] = ri.Interface().(string)
		}
	}
}
