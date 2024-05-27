package u128

import (
	"testing"
)

func Test_sliceToU128(t *testing.T) {
	type args struct {
		buf []byte
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				buf: []byte{},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "too long",
			args: args{
				buf: make([]byte, 17),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				buf: []byte{5, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0},
			},
			want:    "129127208515966861317",
			wantErr: false,
		},
		{
			name: "valid 2",
			args: args{
				buf: []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			},
			want:    "340282366920938463463374607431768211455",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SliceToU128(tt.args.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("sliceToU128() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if err != nil {
				return
			}

			if got.String() != tt.want {
				t.Errorf("sliceToU128() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}
