package chunk

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
	"github.com/stretchr/objx"
)

func WriteChunk(d Region[map[string]string]) error {
	rg, err := region.Open(d.FilePath)
	if err != nil {
		return fmt.Errorf("WriteChunk: %w", err)
	}
	defer rg.Close()

	for _, v := range d.Chunk {
		if !rg.ExistSector(v.X, v.Z) {
			return fmt.Errorf("WriteChunk: %w", ErrNotExistSector{
				X:        v.X,
				Z:        v.Z,
				FilePath: d.FilePath,
			})

		}
		b, err := rg.ReadSector(v.X, v.Z)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		b, err = mcDecompress(b)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		var a map[string]any
		err = nbt.Unmarshal(b, &a)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		err = merge(&a, v.Data)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		bb, err := nbt.Marshal(a)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		bb, err = mcEncompress(bb)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
		err = rg.WriteSector(v.X, v.Z, bb)
		if err != nil {
			return fmt.Errorf("WriteChunk: %w", err)
		}
	}
	return nil
}

var reg = regexp.MustCompile(`\[(\d*)\]$`)

func merge(dst *map[string]any, src map[string]string) error {
	m := objx.New(*dst)
	for k, v := range src {
		if strings.HasSuffix(k, "]") {
			sl := reg.FindStringSubmatch(k)
			i, err := strconv.Atoi(sl[1])
			if err != nil {
				return fmt.Errorf("merge: %w", err)
			}
			nk := reg.ReplaceAllString(k, "")
			data := m.Get(nk).Data()
			err = setList(&data, i, v)
			if err != nil {
				return fmt.Errorf("merge: %w", err)
			}
			m = m.Set(nk, data)
			continue
		}
		m = m.Set(k, v)
	}
	*dst = (map[string]any)(m)
	return nil
}

func setList(data any, index int, v any) error {
	vv := reflect.ValueOf(data)
	for vv.Kind() == reflect.Pointer || vv.Kind() == reflect.Interface {
		vv = vv.Elem()
	}
	if index >= vv.Len() {
		return fmt.Errorf("setList: %w", ErrOutRange)
	}
	vv.Index(index).Set(reflect.ValueOf(v))
	return nil
}

var ErrOutRange = errors.New("超过数组范围")

type ErrNotExistSector struct {
	X        int
	Z        int
	FilePath string
}

func (e ErrNotExistSector) Error() string {
	return fmt.Sprintf("没在 %v 中找到 %v %v", e.FilePath, e.X, e.Z)
}

var ErrNotMap = errors.New("not map")
