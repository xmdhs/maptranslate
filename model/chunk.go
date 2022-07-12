package model

type NbtHasText struct {
	Level         Level       `json:",omitempty"`
	BlockEntities interface{} `nbt:"block_entities" json:"block_entities,omitempty"`
	Entities      interface{} `nbt:"Entities" json:"Entities,omitempty"`
}

type Level struct {
	Entities     interface{} `nbt:"Entities" json:"Entities,omitempty"`
	TileEntities interface{} `nbt:"TileEntities" json:"TileEntities,omitempty"`
}
