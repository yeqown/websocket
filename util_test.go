package websocket

import "testing"

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
			_ = got
		})
	}
}
