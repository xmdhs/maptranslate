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

func WriteChunk(d Region[[]Entities]) error {
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

func merge(dst *map[string]any, src []Entities) error {
	m := objx.New(*dst)
	for _, v := range src {
		al, ok := m.Get(v.Root).Data().([]any)
		if !ok {
			return fmt.Errorf("merge: %w", ErrNotSlice)
		}
		for i, vv := range al {
			if !anyEqEntities(vv, v) {
				continue
			}
			tm, ok := vv.(map[string]any)
			if !ok {
				return fmt.Errorf("merge: %w", ErrNotMap)
			}
			err := mergeMap(&tm, v.PATH)
			if err != nil {
				return fmt.Errorf("merge: %w", err)
			}
			al[i] = tm
		}
		m.Set(v.Root, al)
	}
	*dst = (map[string]any)(m)
	return nil
}

func mergeMap(dst *map[string]any, src map[string]string) error {
	m := objx.New(*dst)
	for k, v := range src {
		if strings.HasSuffix(k, "]") {
			sl := reg.FindStringSubmatch(k)
			i, err := strconv.Atoi(sl[1])
			if err != nil {
				return fmt.Errorf("mergeMap: %w", err)
			}
			nk := reg.ReplaceAllString(k, "")
			data := m.Get(nk).Data()
			err = setList(&data, i, v)
			if err != nil {
				return fmt.Errorf("mergeMap: %w", err)
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
	if vv.Kind() != reflect.Slice {
		return fmt.Errorf("setList: %w", ErrNotSlice)
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

func anyEqEntities(a any, e Entities) bool {
	iseq := false

	m, ok := a.(map[string]any)
	if !ok {
		return false
	}

	if v, ok := m["x"]; ok {
		x, ok := v.(int32)
		if !ok {
			return false
		}
		iseq = e.POS[0] == int(x)
	}
	if v, ok := m["y"]; ok && iseq {
		y, ok := v.(int32)
		if !ok {
			return false
		}
		iseq = e.POS[1] == int(y)
	}
	if v, ok := m["z"]; ok && iseq {
		z, ok := v.(int32)
		if !ok {
			return false
		}
		iseq = e.POS[2] == int(z)
	}
	if iseq {
		return iseq
	}

	if e.UUID != "" {
		uuid, err := getUUIDformap(m)
		if err != nil {
			panic(err)
		}
		return uuid == e.UUID
	}
	return false
}
