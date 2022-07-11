package model

type NbtHasText struct {
	Entities      interface{} `nbt:"Entities" json:"Entities,omitempty"`
	BlockEntities interface{} `nbt:"block_entities" json:"block_entities,omitempty"`
	Entities1     interface{} `nbt:"entities" json:"entities,omitempty"`
	TileEntities  interface{} `nbt:"TileEntities" json:"TileEntities,omitempty"`
}
