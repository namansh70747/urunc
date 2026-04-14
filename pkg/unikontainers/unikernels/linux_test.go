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
			assert.NotContains(t, conf, "RLIMIT:", "expected no RLIMIT lines")
		})
	}
}

func TestBuildUrunitConfigSingleRlimit(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	assert.Contains(t, conf, "RLIMIT:RLIMIT_NOFILE:1024:4096\n")
}

func TestBuildUrunitConfigMultipleRlimits(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
		{Type: "RLIMIT_NPROC", Soft: 512, Hard: 1024},
		{Type: "RLIMIT_AS", Soft: 0, Hard: 0},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	assert.Contains(t, conf, "RLIMIT:RLIMIT_NOFILE:1024:4096\n")
	assert.Contains(t, conf, "RLIMIT:RLIMIT_NPROC:512:1024\n")
	assert.Contains(t, conf, "RLIMIT:RLIMIT_AS:0:0\n")
}

func TestBuildUrunitConfigRlimitsInsideProcessBlock(t *testing.T) {
	l := newTestLinux([]specs.POSIXRlimit{
		{Type: "RLIMIT_NOFILE", Soft: 1024, Hard: 4096},
	})
	conf := l.buildUrunitConfig()
	t.Logf("generated urunit.conf:\n%s", conf)

	ucs := strings.Index(conf, "UCS\n")
	uce := strings.Index(conf, "UCE\n")
	if ucs < 0 || uce < 0 || ucs >= uce {
		t.Fatalf("invalid UCS/UCE markers:\n%s", conf)
	}

	block := conf[ucs : uce+4]
	assert.Contains(t, block, "RLIMIT:RLIMIT_NOFILE:1024:4096\n", "expected RLIMIT line inside process block")
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
