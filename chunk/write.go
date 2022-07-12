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
				panic(err)
			}
			nk := reg.ReplaceAllString(k, "")
			data := m.Get(nk).Data()
			setList(&data, i, v)
			m = m.Set(nk, data)
			continue
		}
		m = m.Set(k, v)
	}
	*dst = (map[string]any)(m)
	return nil
}

func setList(data any, index int, v any) {
	vv := reflect.ValueOf(data).Elem().Elem()
	vv.Index(index).Set(reflect.ValueOf(v))
}

type ErrNotExistSector struct {
	X        int
	Z        int
	FilePath string
}

func (e ErrNotExistSector) Error() string {
	return fmt.Sprintf("没在 %v 中找到 %v %v", e.FilePath, e.X, e.Z)
}

func delKey(m *map[string]any) {
	delKeyL := []string{}
	for k, v := range *m {
		if v == nil {
			delKeyL = append(delKeyL, k)
		}
		if mm, ok := v.(map[string]any); ok {
			delKey(&mm)
			if len(mm) == 0 {
				delKeyL = append(delKeyL, k)
			}
			(*m)[k] = mm
		}
	}
	for _, v := range delKeyL {
		delete(*m, v)
	}

}

var ErrNotMap = errors.New("not map")
