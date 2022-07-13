package chunk

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func PaseMca[K any](f io.ReadWriteSeeker, filePath string) (Region[[]Entities], error) {
	rg, err := region.Load(f)
	if err != nil {
		return Region[[]Entities]{}, fmt.Errorf("PaseMca: %w", err)
	}
	defer rg.Close()
	cl := make([]Chunk[[]Entities], 0)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			if !rg.ExistSector(x, y) {
				continue
			}
			b, err := rg.ReadSector(x, y)
			if err != nil {
				return Region[[]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			b, err = mcDecompress(b)
			if err != nil {
				return Region[[]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			var v K
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return Region[[]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			el, err := getStrPath(v, "")
			if err != nil {
				return Region[[]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			cl = append(cl, Chunk[[]Entities]{
				X:    x,
				Z:    y,
				Data: el,
			})
		}
	}
	return Region[[]Entities]{
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

func getStrPath(v any, path string) ([]Entities, error) {
	dt := reflect.TypeOf(v)
	dv := reflect.ValueOf(v)
	if dv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("getStrPath: %w", ErrNotStruct)
	}
	l := reflect.VisibleFields(dt)
	sl := make([]Entities, 0)

	for _, t := range l {
		v := dv.FieldByIndex(t.Index)
		name := t.Name

		if v, ok := t.Tag.Lookup("json"); ok {
			vv := strings.TrimSuffix(v, ",omitempty")
			if vv != "" {
				v = vv
			}
			name = v
		}

		nPath := ""
		if path == "" {
			nPath = name
		} else {
			nPath = path + "." + name
		}
		if v.Kind() == reflect.Struct {
			m, err := getStrPath(v.Interface(), name)
			if err != nil {
				return nil, err
			}
			sl = append(sl, m...)
			continue
		}

		if v.Kind() != reflect.Interface || v.IsNil() {
			continue
		}
		v = v.Elem()
		if v.Kind() != reflect.Slice {
			return nil, fmt.Errorf("getStrPath: %w", ErrNotSlice)
		}
		vlen := v.Len()
		for i := 0; i < vlen; i++ {
			m, ok := mapify(v.Index(i).Interface())
			if !ok {
				return nil, fmt.Errorf("getStrPath: %w", ErrNotMap)
			}
			v, err := getEntitiesForMap(m, nPath)
			if err != nil {
				return nil, fmt.Errorf("getStrPath: %w", err)
			}
			sl = append(sl, v)
		}

	}
	return sl, nil
}

func getStrPathMap(m map[string]any, path string, sm *map[string]string) {
	for k, v := range m {
		npath := k
		if path != "" {
			npath = path + "." + k
		}
		rt := reflect.TypeOf(v)
		rk := rt.Kind()
		switch rk {
		case reflect.Slice:
			getStrPathSlice(v, npath, sm)
		case reflect.Map:
			getStrPathMap(v.(map[string]any), npath, sm)
		case reflect.String:
			(*sm)[npath] = v.(string)
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

type Entities struct {
	UUID string            `json:",omitempty"`
	POS  []int             `json:",omitempty"`
	PATH map[string]string `json:",omitempty"`
	Root string            `json:",omitempty"`
}

func getEntitiesForMap(m map[string]any, path string) (Entities, error) {
	e := Entities{}

	uuid, err := getUUIDformap(m)
	if err != nil {
		return Entities{}, fmt.Errorf("getEntitiesForMap: %w", err)
	}
	e.UUID = uuid

	xv, xok := m["x"]
	if xok {
		e.POS = make([]int, 3)
		i, ok := xv.(int32)
		if !ok {
			return Entities{}, fmt.Errorf("getEntitiesForMap: %w", ErrNotInt32)
		}
		e.POS[0] = int(i)
	}
	yv, yok := m["y"]
	if yok {
		i, ok := yv.(int32)
		if !ok {
			return Entities{}, fmt.Errorf("getEntitiesForMap: %w", ErrNotInt32)
		}
		e.POS[1] = int(i)
	}
	zv, zok := m["z"]
	if zok {
		i, ok := zv.(int32)
		if !ok {
			return Entities{}, fmt.Errorf("getEntitiesForMap: %w", ErrNotInt32)
		}
		e.POS[2] = int(i)
	}
	if xok != yok || zok != xok || yok != zok {
		return Entities{}, fmt.Errorf("getEntitiesForMap: %w", ErrPos)
	}

	sm := make(map[string]string)

	getStrPathMap(m, "", &sm)
	e.PATH = sm
	e.Root = path

	return e, nil
}

var (
	ErrNBTUUID   = errors.New("错误的 uuid")
	ErrNotStruct = errors.New("不是 struct")
	ErrNotSlice  = errors.New("不是 slice")
	ErrNotInt32  = errors.New("不是 int32")
	ErrPos       = errors.New("错误的坐标")
)

func getUUIDformap(m map[string]any) (string, error) {
	uuid := ""
	if v, ok := m["UUID"]; ok {
		bw := &bytes.Buffer{}
		i32l, ok := v.([]int32)
		if !ok {
			return "", fmt.Errorf("getUUIDformap: %w", ErrNotSlice)
		}
		err := binary.Write(bw, binary.BigEndian, i32l)
		if err != nil {
			return "", fmt.Errorf("getUUIDformap: %w", err)
		}
		uuid = hex.EncodeToString(bw.Bytes())
	}

	if ul, ok := m["UUIDLeast"]; ok {
		um, ok := m["UUIDMost"]
		if !ok {
			return "", fmt.Errorf("getUUIDformap: %w", ErrNBTUUID)
		}
		bw := &bytes.Buffer{}
		err := binary.Write(bw, binary.BigEndian, um)
		if err != nil {
			return "", fmt.Errorf("getUUIDformap: %w", err)
		}
		err = binary.Write(bw, binary.BigEndian, ul)
		if err != nil {
			return "", fmt.Errorf("getUUIDformap: %w", err)
		}
		uuid = hex.EncodeToString(bw.Bytes())
	}
	return uuid, nil
}
