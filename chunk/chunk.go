package chunk

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func PaseMca[K any](f io.ReadWriteSeeker) ([]K, error) {
	rg, err := region.Load(f)
	if err != nil {
		return nil, fmt.Errorf("PaseMca: %w", err)
	}
	cl := make([]K, 0)
	for x := 0; x < 32; x++ {
		for y := 0; y < 32; y++ {
			if !rg.ExistSector(x, y) {
				continue
			}
			b, err := rg.ReadSector(x, y)
			if err != nil {
				return nil, fmt.Errorf("PaseMca: %w", err)
			}
			b, err = mcDecompress(b)
			if err != nil {
				return nil, fmt.Errorf("PaseMca: %w", err)
			}
			var v K
			err = nbt.Unmarshal(b, &v)
			if err != nil {
				return nil, fmt.Errorf("PaseMca: %w", err)
			}
			cl = append(cl, v)
		}
	}
	return cl, nil

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

var (
	ErrInvalidChunk       = errors.New("invalid chunk")
	ErrUnKnownCompression = errors.New("unknown compression")
)
