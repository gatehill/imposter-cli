package stringutil

import "testing"

func TestGetMatchingSuffix(t *testing.T) {
	type args struct {
		entry    string
		suffixes []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no suffixes",
			args: args{
				entry:    "foo",
				suffixes: []string{},
			},
			want: "",
		},
		{
			name: "no match",
			args: args{
				entry:    "foo",
				suffixes: []string{".bar", ".qux"},
			},
			want: "",
		},
		{
			name: "match",
			args: args{
				entry:    "foo.bar",
				suffixes: []string{".bar", ".qux"},
			},
			want: ".bar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMatchingSuffix(tt.args.entry, tt.args.suffixes); got != tt.want {
				t.Errorf("GetMatchingSuffix() = %v, want %v", got, tt.want)
			}
		})
	}
}
