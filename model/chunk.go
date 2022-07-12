package model

type NbtHasText struct {
	Level         level
	BlockEntities interface{} `nbt:"block_entities" json:"block_entities,omitempty"`
	Entities      interface{} `nbt:"entities" json:"entities,omitempty"`
}

type level struct {
	Entities     interface{} `nbt:"Entities" json:"Entities,omitempty"`
	TileEntities interface{} `nbt:"TileEntities" json:"TileEntities,omitempty"`
}
