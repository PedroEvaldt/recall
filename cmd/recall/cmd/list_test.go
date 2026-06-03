package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestListPreRunE(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "No args and no --all",
			args:    []string{"list"},
			wantErr: "either provide a query or use --all",
		},
		{
			name:    "--all flag with query",
			args:    []string{"list", "--all", "go"},
			wantErr: "--all cannot be combined",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetState(t)
			gotOutput := new(bytes.Buffer)
			gotErr := new(bytes.Buffer)
			rootCmd.SetArgs(tt.args)
			rootCmd.SetOut(gotOutput)
			rootCmd.SetErr(gotErr)
			err := rootCmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q; got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q; got %v", tt.wantErr, err.Error())
			}
		})
	}
}

func TestBuildListParams(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		all          bool
		limit        int
		wantQuery    string
		wantLimitStr string
	}{
		{
			name:         "query with multiple words",
			args:         []string{"go", "struct"},
			all:          false,
			limit:        0,
			wantQuery:    "go struct",
			wantLimitStr: "",
		},
		{
			name:         "all flag, no query",
			args:         []string{},
			all:          true,
			limit:        0,
			wantQuery:    "",
			wantLimitStr: "",
		},
		{
			name:         "query with limit",
			args:         []string{"go"},
			all:          false,
			limit:        5,
			wantQuery:    "go",
			wantLimitStr: "5",
		},
		{
			name:         "all with limit",
			args:         []string{},
			all:          true,
			limit:        10,
			wantQuery:    "",
			wantLimitStr: "10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotLimitStr := buildListParams(tt.args, tt.all, tt.limit)
			if gotQuery != tt.wantQuery {
				t.Errorf("query: expected %q; got %q", tt.wantQuery, gotQuery)
			}
			if gotLimitStr != tt.wantLimitStr {
				t.Errorf("limitStr: expected %q; got %q", tt.wantLimitStr, gotLimitStr)
			}
		})
	}
}
