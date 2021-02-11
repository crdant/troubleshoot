package collect

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

type BlockDeviceInfo struct {
	Name             string `json:"name"`
	KernelName       string `json:"kernel_name"`
	ParentKernelName string `json:"parent_kernel_name"`
	Type             string `json:"type"`
	Major            int    `json:"major"`
	Minor            int    `json:"minor"`
	Size             uint64 `json:"size"`
	FilesystemType   string `json:"filesystem_type"`
	Mountpoint       string `json:"mountpoint"`
	Serial           string `json:"serial"`
	ReadOnly         bool   `json:"read_only"`
	Removable        bool   `json:"removable"`
}

const lsblkColumns = "NAME,KNAME,PKNAME,TYPE,MAJ:MIN,SIZE,FSTYPE,MOUNTPOINT,SERIAL,RO,RM"
const lsblkFormat = `NAME=%q KNAME=%q PKNAME=%q TYPE=%q MAJ:MIN="%d:%d" SIZE="%d" FSTYPE=%q MOUNTPOINT=%q SERIAL=%q RO="%d" RM="%d0"`

func HostBlockDevices(c *HostCollector) (map[string][]byte, error) {
	var devices []BlockDeviceInfo

	cmd := exec.Command("lsblk", "--noheadings", "--bytes", "--pairs", "-o", lsblkColumns)
	stdout, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute lsblk")
	}
	buf := bytes.NewBuffer(stdout)
	scanner := bufio.NewScanner(buf)

	for scanner.Scan() {
		bdi := BlockDeviceInfo{}
		var ro int
		var rm int
		fmt.Sscanf(
			scanner.Text(),
			lsblkFormat,
			&bdi.Name,
			&bdi.KernelName,
			&bdi.ParentKernelName,
			&bdi.Type,
			&bdi.Major,
			&bdi.Minor,
			&bdi.Size,
			&bdi.FilesystemType,
			&bdi.Mountpoint,
			&bdi.Serial,
			&ro,
			&rm,
		)
		bdi.ReadOnly = ro == 1
		bdi.Removable = rm == 1

		devices = append(devices, bdi)
	}

	b, err := json.Marshal(devices)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal block device info")
	}

	return map[string][]byte{
		"system/block_devices.json": b,
	}, nil
}
