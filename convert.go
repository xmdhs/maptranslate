package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/xmdhs/maptranslate/chunk"
)

type Lang struct {
	SHA256 string
	TEXT   string
}

func regionList2LangList(l []chunk.Region[[]chunk.Entities]) []Lang {
	ll := make([]Lang, 0)
	set := make(map[string]struct{})

	for _, v := range l {
		for _, c := range v.Chunk {
			for _, d := range c.Data {
				for _, v := range d.PATH {
					if _, ok := set[v]; ok {
						continue
					}
					set[v] = struct{}{}
					h := sha256.New()
					h.Write([]byte(v))

					ll = append(ll, Lang{
						SHA256: hex.EncodeToString(h.Sum(nil)),
						TEXT:   v,
					})
				}
			}
		}
	}
	return ll
}

func useLangList(l []Lang, cl *[]chunk.Region[[]chunk.Entities]) {
	m := make(map[string]string)
	for _, v := range l {
		m[v.SHA256] = v.TEXT
	}

	for i := range *cl {
		for c := range (*cl)[i].Chunk {
			for d := range (*cl)[i].Chunk[c].Data {
				for k, v := range (*cl)[i].Chunk[c].Data[d].PATH {
					h := sha256.New()
					h.Write([]byte(v))
					s256 := hex.EncodeToString(h.Sum(nil))
					if v, ok := m[s256]; ok {
						(*cl)[i].Chunk[c].Data[d].PATH[k] = v
					}
				}
			}
		}
	}
}
