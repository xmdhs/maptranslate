package model

type NbtHasText struct {
	Level         Level       `json:"Level,omitempty"`
	BlockEntities interface{} `nbt:"block_entities" json:"block_entities,omitempty"`
	Entities      interface{} `nbt:"Entities" json:"Entities,omitempty"`
}

type Level struct {
	Entities     interface{} `nbt:"Entities" json:"Entities,omitempty"`
	TileEntities interface{} `nbt:"TileEntities" json:"TileEntities,omitempty"`
}

type NbtPosUUID struct {
	Level         LevelPosUUID `json:"Level,omitempty"`
	BlockEntities []PosUUID    `nbt:"block_entities" json:"block_entities,omitempty"`
	Entities      []PosUUID    `nbt:"Entities" json:"Entities,omitempty"`
}

type LevelPosUUID struct {
	Entities     []PosUUID `nbt:"Entities" json:"Entities,omitempty"`
	TileEntities []PosUUID `nbt:"TileEntities" json:"TileEntities,omitempty"`
}

type PosUUID struct {
	X         int `nbt:"x" json:"x"`
	Y         int `nbt:"y" json:"y"`
	Z         int `nbt:"z" json:"z"`
	UUID      []int32
	UUIDLeast int64
	UUIDMost  int64
}
