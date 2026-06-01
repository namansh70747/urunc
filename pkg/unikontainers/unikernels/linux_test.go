// Copyright (c) 2023-2026, Nubificus LTD
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unikernels

import (
	"strings"
	"testing"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
)

func newTestLinux(rlimits []specs.POSIXRlimit) *Linux {
	return &Linux{
		Env:     []string{"PATH=/usr/local/bin"},
		Monitor: "qemu",
		ProcConfig: types.ProcessConfig{
			UID:     1000,
			GID:     1000,
			WorkDir: "/app",
			Rlimits: rlimits,
		},
	}
}

func TestBuildUrunitConfigNoRlimits(t *testing.T) {
	tests := []struct {
		name    string
		rlimits []specs.POSIXRlimit
	}{
		{name: "nil", rlimits: nil},
		{name: "empty", rlimits: []specs.POSIXRlimit{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newTestLinux(tt.rlimits)
			conf := l.buildUrunitConfig()
			t.Logf("generated urunit.conf:\n%s", conf)
			// With no rlimits the whole RLS block is omitted, so the
			// generated configuration stays identical to guests that
			// predate the rlimit support.
			assert.NotContains(t, conf, "RLS\n", "expected no rlimit block")
			assert.NotContains(t, conf, "RLE\n", "expected no rlimit block")
			assert.NotContains(t, conf, "TYPE:", "expected no rlimit entries")
		})
	}
}

func TestBuildUrunitConfigSingleRlimit(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	assert.Contains(t, conf, "RLS\n")
	assert.Contains(t, conf, "NUM:1\n")
	assert.Contains(t, conf, "TYPE:RLIMIT_NOFILE\n")
	assert.Contains(t, conf, "SOFT:1024\n")
	assert.Contains(t, conf, "HARD:4096\n")
	assert.Contains(t, conf, "RLE\n")
}

func TestBuildUrunitConfigMultipleRlimits(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
		{Type: "RLIMIT_NPROC", Soft: 512, Hard: 1024},
		{Type: "RLIMIT_AS", Soft: 0, Hard: 0},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	assert.Contains(t, conf, "NUM:3\n", "NUM must match the number of entries")
	assert.Contains(t, conf, "TYPE:RLIMIT_NOFILE\nSOFT:1024\nHARD:4096\n")
	assert.Contains(t, conf, "TYPE:RLIMIT_NPROC\nSOFT:512\nHARD:1024\n")
	assert.Contains(t, conf, "TYPE:RLIMIT_AS\nSOFT:0\nHARD:0\n")
}

func TestBuildUrunitConfigRlimitsAreInOwnBlock(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	ucs := strings.Index(conf, "UCS\n")
	uce := strings.Index(conf, "UCE\n")
	rls := strings.Index(conf, "RLS\n")
	rle := strings.Index(conf, "RLE\n")
	ubs := strings.Index(conf, "UBS\n")
	if ucs < 0 || uce < 0 || rls < 0 || rle < 0 || ubs < 0 {
		t.Fatalf("missing one of UCS/UCE/RLS/RLE/UBS markers:\n%s", conf)
	}

	// The rlimit entries must live in their own RLS..RLE block and not leak
	// into the UCS..UCE process-config block.
	procBlock := conf[ucs : uce+len("UCE\n")]
	assert.NotContains(t, procBlock, "TYPE:", "rlimits must not be inside the process block")

	// Block ordering must be UCS..UCE, then RLS..RLE, then UBS, so urunit
	// parses each block in the expected sequence.
	assert.Less(t, uce, rls, "RLS block must come after the UCE marker")
	assert.Less(t, rls, rle, "RLS must precede RLE")
	assert.Less(t, rle, ubs, "RLS block must come before the UBS block")
}

func TestBuildUrunitConfigUIDGIDWorkdir(t *testing.T) {
	l := &Linux{
		ProcConfig: types.ProcessConfig{
			UID:     500,
			GID:     501,
			WorkDir: "/workdir",
		},
	}
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	assert.Contains(t, conf, "UID:500\n")
	assert.Contains(t, conf, "GID:501\n")
	assert.Contains(t, conf, "WD:/workdir\n")
}
