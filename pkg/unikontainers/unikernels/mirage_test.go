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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
)

func TestMirageInitSubnetMask(t *testing.T) {
	tests := []struct {
		name        string
		mask        string
		ip          string
		gateway     string
		wantAddress string
		wantGateway string
	}{
		{
			name:        "non-/24 mask is used correctly",
			mask:        "255.255.255.240",
			ip:          "10.0.0.1",
			gateway:     "10.0.0.14",
			wantAddress: "--ipv4=10.0.0.1/28",
			wantGateway: "--ipv4-gateway=10.0.0.14",
		},
		{
			name:        "/24 mask still works",
			mask:        "255.255.255.0",
			ip:          "192.168.1.5",
			gateway:     "192.168.1.1",
			wantAddress: "--ipv4=192.168.1.5/24",
			wantGateway: "--ipv4-gateway=192.168.1.1",
		},
		{
			name:        "no network when mask is empty",
			mask:        "",
			ip:          "",
			gateway:     "",
			wantAddress: "",
			wantGateway: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMirage()
			params := types.UnikernelParams{
				Net: types.NetDevParams{
					IP:      tt.ip,
					Mask:    tt.mask,
					Gateway: tt.gateway,
				},
			}
			err := m.Init(params)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantAddress, m.Net.Address)
			assert.Equal(t, tt.wantGateway, m.Net.Gateway)
		})
	}
}

func TestMirageInitBlockIDs(t *testing.T) {
	// Init must preserve the device ID and host path of every block device
	// passed through UnikernelParams, so that MonitorBlockCli can later attach
	// them with the correct Solo5 names.
	m := newMirage()
	params := types.UnikernelParams{
		Block: []types.BlockDevParams{
			{ID: "storage", Source: "/dev/vda"},
			{ID: "data", Source: "/dev/vdb"},
		},
	}
	err := m.Init(params)
	assert.NoError(t, err)
	assert.Equal(t, []MirageBlock{
		{ID: "storage", HostPath: "/dev/vda"},
		{ID: "data", HostPath: "/dev/vdb"},
	}, m.Block)
}

func TestMirageMonitorBlockCli(t *testing.T) {
	tests := []struct {
		name     string
		monitor  string
		blocks   []MirageBlock
		wantArgs []types.MonitorBlockArgs
	}{
		{
			name:     "no blocks returns nil",
			monitor:  "hvt",
			blocks:   nil,
			wantArgs: nil,
		},
		{
			name:    "single block uses its device name",
			monitor: "hvt",
			blocks:  []MirageBlock{{ID: "mydata", HostPath: "/dev/vda"}},
			wantArgs: []types.MonitorBlockArgs{
				{ID: "mydata", Path: "/dev/vda"},
			},
		},
		{
			name:    "empty device name falls back to storage",
			monitor: "hvt",
			blocks:  []MirageBlock{{ID: "", HostPath: "/dev/vda"}},
			wantArgs: []types.MonitorBlockArgs{
				{ID: MirageDefaultBlkID, Path: "/dev/vda"},
			},
		},
		{
			name:    "generic rootfs id falls back to storage",
			monitor: "hvt",
			blocks:  []MirageBlock{{ID: genericRootfsBlkID, HostPath: "/dev/vda"}},
			wantArgs: []types.MonitorBlockArgs{
				{ID: MirageDefaultBlkID, Path: "/dev/vda"},
			},
		},
		{
			name:    "multiple blocks are all attached with their names",
			monitor: "spt",
			blocks: []MirageBlock{
				{ID: "storage", HostPath: "/dev/vda"},
				{ID: "data", HostPath: "/dev/vdb"},
			},
			wantArgs: []types.MonitorBlockArgs{
				{ID: "storage", Path: "/dev/vda"},
				{ID: "data", Path: "/dev/vdb"},
			},
		},
		{
			name:     "unsupported monitor returns nil",
			monitor:  "qemu",
			blocks:   []MirageBlock{{ID: "storage", HostPath: "/dev/vda"}},
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMirage()
			m.Monitor = tt.monitor
			m.Block = tt.blocks
			assert.Equal(t, tt.wantArgs, m.MonitorBlockCli())
		})
	}
}
