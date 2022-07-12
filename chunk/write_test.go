package chunk

import (
	"fmt"
	"testing"
)

func Test_merge(t *testing.T) {
	m := map[string]interface{}{
		"x":     "11",
		"tecc":  12,
		"Level": "www",
	}
	type Level struct {
		Entities     interface{} `nbt:"Entities" json:"Entities,omitempty"`
		TileEntities interface{} `nbt:"TileEntities" json:"TileEntities,omitempty"`
	}
	type NbtHasText struct {
		Level         Level
		BlockEntities interface{} `nbt:"block_entities" json:"block_entities,omitempty"`
		Entities      interface{} `nbt:"entities" json:"entities,omitempty"`
	}

	err := merge(&m, NbtHasText{
		Level: Level{
			Entities: []string{"a", "b"},
		},
		Entities: []int{1, 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(m)
}
