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
	"fmt"
	"strings"

	"github.com/urunc-dev/urunc/pkg/unikontainers/types"
)

const MirageUnikernel string = "mirage"

type Mirage struct {
	Command string
	Monitor string
	Net     MirageNet
	Block   []MirageBlock
}

type MirageNet struct {
	Address string
	Gateway string
}

type MirageBlock struct {
	ID       string
	HostPath string
}

func (m *Mirage) CommandString() (string, error) {
	return fmt.Sprintf("%s %s %s", m.Net.Address,
		m.Net.Gateway,
		m.Command), nil
}

func (m *Mirage) SupportsBlock() bool {
	return true
}

func (m *Mirage) SupportsFS(_ string) bool {
	return false
}

func (m *Mirage) MonitorNetCli(ifName string, mac string) string {
	switch m.Monitor {
	case "hvt", "spt":
		netOption := "--net:service=" + ifName
		netOption += " --net-mac:service=" + mac
		return netOption
	default:
		return ""
	}
}

// MirageDefaultBlkID is the device name (Solo5 manifest name) used for a
// MirageOS block device when the image does not specify one through the
// com.urunc.unikernel.blkDev annotation. It matches the name commonly used by
// MirageOS unikernels and preserves backwards compatibility with images that
// predate the annotation.
const MirageDefaultBlkID string = "storage"

// genericRootfsBlkID is the urunc-internal default ID assigned to the rootfs
// block device (see blockRootfs.getBlockDevs). It is not a MirageOS Solo5
// manifest name, so for MirageOS we treat it the same as an unset name and
// fall back to MirageDefaultBlkID.
const genericRootfsBlkID string = "rootfs"

func (m *Mirage) MonitorBlockCli() []types.MonitorBlockArgs {
	if len(m.Block) == 0 {
		return nil
	}
	switch m.Monitor {
	case "hvt", "spt":
		// Solo5 attaches block devices by a name that the guest is aware of
		// at build time (stored in the Solo5 manifest). urunc obtains this
		// name from the com.urunc.unikernel.blkDev annotation, which the image
		// builder sets to match the unikernel. When the annotation is absent,
		// the ID is either empty or the urunc-internal "rootfs" default, in
		// which case we fall back to the historical MirageOS default
		// ("storage") so that existing single-block images keep working.
		args := make([]types.MonitorBlockArgs, 0, len(m.Block))
		for _, blk := range m.Block {
			id := blk.ID
			if id == "" || id == genericRootfsBlkID {
				id = MirageDefaultBlkID
			}
			args = append(args, types.MonitorBlockArgs{
				ID:   id,
				Path: blk.HostPath,
			})
		}
		return args
	default:
		return nil
	}
}

func (m *Mirage) MonitorCli() types.MonitorCliArgs {
	return types.MonitorCliArgs{}
}

func (m *Mirage) Init(data types.UnikernelParams) error {
	// if Mask is empty, there is no network support
	if data.Net.Mask != "" {
		mask, err := subnetMaskToCIDR(data.Net.Mask)
		if err != nil {
			return err
		}
		m.Net.Address = fmt.Sprintf("--ipv4=%s/%d", data.Net.IP, mask)
		m.Net.Gateway = "--ipv4-gateway=" + data.Net.Gateway
	}
	m.Block = make([]MirageBlock, 0, len(data.Block))
	for _, blk := range data.Block {
		newBlk := MirageBlock{
			ID:       blk.ID,
			HostPath: blk.Source,
		}
		m.Block = append(m.Block, newBlk)
	}

	m.Command = strings.Join(data.CmdLine, " ")
	m.Monitor = data.Monitor

	return nil
}

func newMirage() *Mirage {
	mirageStruct := new(Mirage)
	return mirageStruct
}
