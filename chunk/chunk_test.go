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
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "1",
			args: args{
				v: model.NbtHasText{
					Level: model.Level{
						Entities: map[string]any{
							"aaa": 12,
							"bbb": "bbb",
							"map": map[string]any{
								"ccc": 42,
								"ddd": "42",
							},
						},
					},
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
			},
			want: map[string]string{
				"Level.Entities.bbb":     "bbb",
				"Level.Entities.map.ddd": "42",
				"Entities[0].sss":        "42",
				"Entities[1].3333":       "42",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStrPath(tt.args.v, tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getStrPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
