package main

import (
	"reflect"
	"testing"
)

func TestURLRedacting(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "https://user:pass@repo.com/org/project.git",
			want: "https://...@repo.com/org/project.git",
		},
		{
			url:  "https://github.com/org/project.git",
			want: "https://github.com/org/project.git",
		},
		{
			url:  "git@github.com:Org/Applications.git",
			want: "git@github.com:Org/Applications.git",
		},
	}

	for _, tt := range tests {
		got := redactURL(tt.url)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("redactURL() = %v, want %v", got, tt.want)
		}
	}
}
