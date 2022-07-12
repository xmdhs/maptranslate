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

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func PaseMca[K any](f io.ReadWriteSeeker, filePath string) (Region[K], error) {
	rg, err := region.Load(f)
	if err != nil {
		return Region[K]{}, fmt.Errorf("PaseMca: %w", err)
	}
	defer rg.Close()
	cl := make([]Chunk[K], 0)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			if !rg.ExistSector(x, y) {
				continue
			}
			b, err := rg.ReadSector(x, y)
			if err != nil {
				return Region[K]{}, fmt.Errorf("PaseMca: %w", err)
			}
			b, err = mcDecompress(b)
			if err != nil {
				return Region[K]{}, fmt.Errorf("PaseMca: %w", err)
			}
			var v K
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return Region[K]{}, fmt.Errorf("PaseMca: %w", err)
			}
			cl = append(cl, Chunk[K]{
				X:    x,
				Z:    y,
				Data: v,
			})
		}
	}
	return Region[K]{
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
		if !reflect.ValueOf(v.Data).IsZero() {
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

func ChunkRemoveNullSlice(v any) {
	dt := reflect.TypeOf(v)
	if dt.Kind() != reflect.Pointer {
		panic("not Pointer")
	}
	dv := reflect.ValueOf(v).Elem()
	if dv.Kind() != reflect.Struct {
		return
	}
	l := reflect.VisibleFields(dt.Elem())

	needDel := [][]int{}
	for _, t := range l {
		v := dv.FieldByIndex(t.Index)
		if v.Kind() == reflect.Struct {
			ChunkRemoveNullSlice(v.Addr().Interface())
			continue
		}

		if v.Kind() != reflect.Interface || v.IsNil() {
			continue
		}
		v = v.Elem()

		if v.Kind() != reflect.Slice {
			continue
		}
		if v.Len() == 0 {
			needDel = append(needDel, t.Index)
		}
	}
	for _, v := range needDel {
		v := dv.FieldByIndex(v)
		v.Set(reflect.Zero(v.Type()))
	}
}
