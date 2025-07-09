package common_test

import (
	"strings"
	"testing"

	"github.com/BenasB/bx2cloud/internal/cli/common"
)

func TestArgParse_Uint32(t *testing.T) {
	var tests = []struct {
		inArgs  []string
		outArgs []string
		out     uint32
	}{
		{[]string{}, []string{}, 0},
		{[]string{""}, []string{""}, 0},
		{[]string{"1"}, []string{}, 1},
		{[]string{"4321", "foo"}, []string{"foo"}, 4321},
		{[]string{"foo", "bar"}, []string{"foo", "bar"}, 0},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.inArgs, ","), func(t *testing.T) {
			out, _, _ := common.ParseUint32Arg(&tt.inArgs)
			if !arrEqual(tt.inArgs, tt.outArgs) || out != tt.out {
				t.Fatalf(
					"got %q, %d, want %q, %d",
					tt.inArgs,
					out,
					tt.outArgs,
					tt.out,
				)
			}
		})
	}
}

func TestArgParse_Uint32_Chained(t *testing.T) {
	args := []string{"1", "2", "foo"}
	outArgs := []string{"foo"}
	out1, _, _ := common.ParseUint32Arg(&args)
	out2, _, _ := common.ParseUint32Arg(&args)
	if !arrEqual(args, outArgs) || out1 != 1 || out2 != 2 {
		t.Fatalf(
			"got %q, %d, %d, want %q, %d, %d",
			args,
			out1,
			out2,
			outArgs,
			1,
			2,
		)
	}
}

func arrEqual[T comparable](a []T, b []T) bool {
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
