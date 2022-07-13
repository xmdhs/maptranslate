package chunk

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/xmdhs/maptranslate/model"
)

func TestBlockPos2Mca(t *testing.T) {
	type args struct {
		x int
		z int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1",
			args: args{
				x: 0,
				z: 0,
			},
			want: "r.0.0.mca",
		},
		{
			name: "2",
			args: args{
				x: 1555,
				z: 0,
			},
			want: "r.3.0.mca",
		},
		{
			name: "3",
			args: args{
				x: -3242,
				z: 1111,
			},
			want: "r.-7.2.mca",
		},
		{
			name: "4",
			args: args{
				x: -21421414,
				z: 3333150,
			},
			want: "r.-41839.6510.mca",
		},
		{
			name: "5",
			args: args{
				x: -512,
				z: 513,
			},
			want: "r.-1.1.mca",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BlockPos2Mca(tt.args.x, tt.args.z); got != tt.want {
				t.Errorf("BlockPos2Mca() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaseMca(t *testing.T) {
	f, err := os.Open("r.0.-1.mca")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	type test struct {
		Entities interface{}
	}

	data, err := PaseMca[test](f, "")
	if err != nil {
		t.Fatal(err)
	}
	bb, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(bb))
}

func Test_getStrPath(t *testing.T) {
	type args struct {
		v    any
		path string
		pos  model.NbtPosUUID
	}
	tests := []struct {
		name string
		args args
		want map[string]Entities
	}{
		{
			name: "1",
			args: args{
				v: model.NbtHasText{
					Entities: []any{
						map[string]any{
							"ccc": 42,
							"sss": "42",
						},
						map[string]any{
							"ccc":  42,
							"3333": "42",
						},
					},
				},
				path: "",
				pos: model.NbtPosUUID{
					Entities: []model.PosUUID{
						{
							X: 1,
							Y: 2,
							Z: 3,
						},
						{
							X: 2,
							Y: 3,
							Z: 4,
						},
					},
				},
			},
			want: map[string]Entities{
				"Entities[0].sss": {
					UUID: "00000000000000000000000000000000",
					TEXT: "42",
					POS:  [3]int{1, 2, 3},
				},
				"Entities[1].3333": {
					UUID: "00000000000000000000000000000000",
					TEXT: "42",
					POS:  [3]int{2, 3, 4},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStrPath(tt.args.v, tt.args.path, tt.args.pos); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getStrPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_posUUID2Ent(t *testing.T) {
	type args struct {
		pos model.PosUUID
	}
	tests := []struct {
		name string
		args args
		want Entities
	}{
		{
			name: "1",
			args: args{
				pos: model.PosUUID{
					X:         0,
					Y:         0,
					Z:         0,
					UUID:      []int32{},
					UUIDLeast: -6384696206158828554,
					UUIDMost:  -568210367123287600,
				},
			},
			want: Entities{
				UUID: "f81d4fae7dec11d0a76500a0c91e6bf6",
				POS:  [3]int{},
				TEXT: "",
			},
		},
		{
			name: "2",
			args: args{
				pos: model.PosUUID{
					X:    0,
					Y:    0,
					Z:    0,
					UUID: []int32{-1622059206, 1589986690, -1943336457, -259745871},
				},
			},
			want: Entities{
				UUID: "9f51573a5ec545828c2b09f7f08497b1",
				POS:  [3]int{},
				TEXT: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := posUUID2Ent(tt.args.pos); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("posUUID2Ent() = %v, want %v", got, tt.want)
			}
		})
	}
}
