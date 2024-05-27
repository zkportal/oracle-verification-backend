package attestation

import (
	"net/http"
	"reflect"
	"testing"

	encoding "github.com/zkportal/aleo-oracle-encoding"
)

func Test_decodeProofData(t *testing.T) {
	var htmlResultElement = "element"
	var htmlContentType = "text/html"
	var requestBody = "{\"userid\": 123456, \"token\": \"abcdef\"}"

	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *DecodedProofData
		wantErr bool
	}{
		{
			name: "all fields",
			args: args{
				buf: []byte{6, 0, 8, 0, 8, 0, 4, 0, 1, 0, 31, 0, 43, 0, 16, 0, 80, 0, 128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 115, 116, 114, 105, 110, 103, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 183, 47, 112, 101, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 104, 116, 116, 112, 115, 58, 47, 47, 108, 111, 99, 97, 108, 104, 111, 115, 116, 58, 56, 48, 56, 48, 47, 114, 101, 115, 111, 117, 114, 99, 101, 0, 47, 104, 116, 109, 108, 47, 98, 111, 100, 121, 47, 100, 105, 118, 47, 109, 97, 105, 110, 47, 116, 97, 98, 108, 101, 47, 116, 98, 111, 100, 121, 47, 116, 114, 91, 50, 93, 47, 116, 100, 91, 49, 93, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 80, 79, 83, 84, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 16, 0, 75, 101, 101, 112, 45, 65, 108, 105, 118, 101, 58, 102, 97, 108, 115, 101, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 21, 0, 85, 115, 101, 114, 45, 65, 103, 101, 110, 116, 58, 99, 117, 114, 108, 32, 49, 46, 50, 46, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 116, 101, 120, 116, 47, 104, 116, 109, 108, 0, 0, 0, 0, 0, 0, 0, 37, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 117, 115, 101, 114, 105, 100, 34, 58, 32, 49, 50, 51, 52, 53, 54, 44, 32, 34, 116, 111, 107, 101, 110, 34, 58, 32, 34, 97, 98, 99, 100, 101, 102, 34, 125, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			want: &DecodedProofData{
				ResponseStatusCode: 200,
				AttestationData:    "string",
				Timestamp:          1701851063,
				AttestationRequest: AttestationRequest{
					Url:                "https://localhost:8080/resource",
					Selector:           "/html/body/div/main/table/tbody/tr[2]/td[1]",
					RequestMethod:      http.MethodPost,
					ResponseFormat:     "html",
					HTMLResultType:     &htmlResultElement,
					RequestContentType: &htmlContentType,
					RequestHeaders: map[string]string{
						"Keep-Alive": "false",
						"User-Agent": "curl 1.2.3",
					},
					RequestBody: &requestBody,
					EncodingOptions: encoding.EncodingOptions{
						Value: "string",
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeProofData(tt.args.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeProofData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeProofData() = %v, want %v", got, tt.want)
			}
		})
	}
}
