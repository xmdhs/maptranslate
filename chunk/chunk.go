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
	"github.com/xmdhs/maptranslate/model"
)

func PaseMca[K any](f io.ReadWriteSeeker, filePath string) (Region[map[string]Entities], error) {
	rg, err := region.Load(f)
	if err != nil {
		return Region[map[string]Entities]{}, fmt.Errorf("PaseMca: %w", err)
	}
	defer rg.Close()
	cl := make([]Chunk[map[string]Entities], 0)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			if !rg.ExistSector(x, y) {
				continue
			}
			b, err := rg.ReadSector(x, y)
			if err != nil {
				return Region[map[string]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			b, err = mcDecompress(b)
			if err != nil {
				return Region[map[string]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			var v K
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return Region[map[string]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			var pos model.NbtPosUUID
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return Region[map[string]Entities]{}, fmt.Errorf("PaseMca: %w", err)
			}
			cl = append(cl, Chunk[map[string]Entities]{
				X:    x,
				Z:    y,
				Data: getStrPath(v, "", pos),
			})
		}
	}
	return Region[map[string]Entities]{
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

func getStrPath(v any, path string, pos model.NbtPosUUID) map[string]Entities {
	dt := reflect.TypeOf(v)
	dv := reflect.ValueOf(v)
	if dv.Kind() != reflect.Struct {
		panic("nor struct")
	}
	l := reflect.VisibleFields(dt)
	sm := make(map[string]Entities)

	for _, t := range l {
		v := dv.FieldByIndex(t.Index)
		name := t.Name

		if v, ok := t.Tag.Lookup("json"); ok {
			l := strings.Split(v, ",")
			if len(l) != 0 {
				v = l[0]
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
			m := getStrPath(v.Interface(), name, pos)
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
			getStrPathMap(m, nPath, &sm, Entities{}, pos)
		}
		if v.Kind() == reflect.Slice {
			getStrPathSlice(v.Interface(), nPath, &sm, Entities{}, pos)
		}
	}
	return sm
}

func getStrPathMap(m map[string]any, path string, sm *map[string]Entities, e Entities, pos model.NbtPosUUID) {
	for k, v := range m {
		rt := reflect.TypeOf(v)
		rk := rt.Kind()
		switch rk {
		case reflect.Slice:
			getStrPathSlice(v, path+"."+k, sm, e, pos)
		case reflect.Map:
			getStrPathMap(v.(map[string]any), path+"."+k, sm, e, pos)
		case reflect.String:
			e.TEXT = v.(string)
			(*sm)[path+"."+k] = e
		}
	}
}

func getStrPathSlice(l any, path string, sm *map[string]Entities, e Entities, pos model.NbtPosUUID) {
	rl := reflect.ValueOf(l)
	rlen := rl.Len()
	needList := false
	if strings.HasSuffix("Entities", path) || strings.HasSuffix("TileEntities", path) || strings.HasSuffix("block_entities", path) {
		needList = true
	}

	for i := 0; i < rlen; i++ {
		ri := rl.Index(i)
		if ri.Kind() == reflect.Interface {
			ri = ri.Elem()
		}
		if needList {
			switch path {
			case "Level.Entities":
				e = posUUID2Ent(pos.Level.Entities[i])
			case "Level.TileEntities":
				e = posUUID2Ent(pos.Level.TileEntities[i])
			case "block_entities":
				e = posUUID2Ent(pos.BlockEntities[i])
			case "Entities":
				e = posUUID2Ent(pos.Entities[i])
			}
		}
		switch ri.Kind() {
		case reflect.Slice:
			getStrPathSlice(ri.Interface(), path+"["+strconv.Itoa(i)+"]", sm, e, pos)
		case reflect.Map:
			getStrPathMap(ri.Interface().(map[string]any), path+"["+strconv.Itoa(i)+"]", sm, e, pos)
		case reflect.String:
			e.TEXT = ri.Interface().(string)
			(*sm)[path+"["+strconv.Itoa(i)+"]"] = e
		}
	}
}

type Entities struct {
	UUID string
	POS  [3]int
	TEXT string
}

func posUUID2Ent(pos model.PosUUID) Entities {
	e := Entities{}
	e.POS[0] = pos.X
	e.POS[1] = pos.Y
	e.POS[2] = pos.Z
	if len(pos.UUID) != 0 {
		bw := &bytes.Buffer{}
		err := binary.Write(bw, binary.BigEndian, pos.UUID)
		if err != nil {
			panic(err)
		}
		e.UUID = hex.EncodeToString(bw.Bytes())
	} else {
		bw := &bytes.Buffer{}
		err := binary.Write(bw, binary.BigEndian, pos.UUIDMost)
		if err != nil {
			panic(err)
		}
		err = binary.Write(bw, binary.BigEndian, pos.UUIDLeast)
		if err != nil {
			panic(err)
		}
		e.UUID = hex.EncodeToString(bw.Bytes())

	}

	return e
}
