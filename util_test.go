package websocket

import (
	"testing"
)

func Test_generateChallengeKey(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "case 0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateChallengeKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("generateChallengeKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(got)
		})
	}
}

func Test_computeAcceptKey(t *testing.T) {
	type args struct {
		challengeKey string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 0",
			args: args{
				challengeKey: "vsj/Lv1PrpaM3phhuQaCwA==",
			},
			want: "hmjGuAvho4DNj8U4MED02EkkeCY=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeAcceptKey(tt.args.challengeKey); got != tt.want {
				t.Errorf("computeAcceptKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

// func Test_bigendianUint64(t *testing.T) {
// 	type args struct {
// 		v uint64
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want uint16
// 	}{
// 		{
// 			name: "case 0",
// 			args: args{
// 				v: 123 << (64 - 16),
// 			},
// 			want: 123,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := bigendian16BitFromUint64(tt.args.v); got != tt.want {
// 				t.Errorf("bigendianUint64() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
