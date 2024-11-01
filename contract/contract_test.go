package contract

import (
	"reflect"
	"slices"
	"testing"
)

func Test_parseSgxUniqueIdStruct(t *testing.T) {
	type args struct {
		uniqueIdStructString string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				uniqueIdStructString: "{\n  chunk_1: 31929802673692760512905395015836068420u128,\n  chunk_2: 335853521753947303372057454886636012152u128\n}",
			},
			want:    "446a519b3ff301317d7ab2a6d074051878c23c345b3f85e76dbc69141309abfc",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSgxUniqueIdStruct(tt.args.uniqueIdStructString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSgxUniqueIdStruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSgxUniqueIdStruct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseNitroPcrValues(t *testing.T) {
	type args struct {
		nitroPcrStructString string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				nitroPcrStructString: "{\n  pcr_0_chunk_1: 71402194384810807695471133674510927100u128,\n  pcr_0_chunk_2: 161208568844425284329478584127483958658u128,\n  pcr_0_chunk_3: 319153641741947202476283715452178757539u128,\n  pcr_1_chunk_1: 160074764010604965432569395010350367491u128,\n  pcr_1_chunk_2: 139766717364114533801335576914874403398u128,\n  pcr_1_chunk_3: 227000420934281803670652481542768973666u128,\n  pcr_2_chunk_1: 264733590264774658848247826143579120213u128,\n  pcr_2_chunk_2: 334747434232414500511461632767813487886u128,\n  pcr_2_chunk_3: 200411607119746324753107350992173755975u128\n}",
			},
			want: []string{
				"fcc4ced3f4bba7352e289a27fb8fb7358255d6b35abafdc8b4a398c418a44779a377979baa62fc78ef6d89aa6bc11af0",
				"0343b056cd8485ca7890ddd833476d78460aed2aa161548e4e26bedf321726696257d623e8805f3f605946b3d8b0c6aa",
				"55a296be86298ce7d58bf289bad529c70e0d50854b475990d4f8ead2bf02d6fb476e717cc80c057abf7cd0f21cdfc596",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNitroPcrValues(tt.args.nitroPcrStructString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNitroPcrValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseNitroPcrValues()\nGot:\n%v\nWant:\n%v", got, tt.want)
			}
		})
	}
}
