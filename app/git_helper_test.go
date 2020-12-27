package repeater4gcsr

import (
	"testing"
)

func Test_transformProtocol(t *testing.T) {
	type args struct {
		http string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"hoge",
			args{
				"https://bitbucket.org/usadamasa/repeater4gcsr",
			},
			"git@bitbucket.org:/usadamasa/repeater4gcsr.git",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformProtocol(tt.args.http)
			if (err != nil) != tt.wantErr {
				t.Errorf("transformProtocol() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("transformProtocol() got = %v, want %v", got, tt.want)
			}
		})
	}
}
