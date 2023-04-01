package reghive

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Device struct {
	GPT    bool   `json:"gpt,omitempty"`
	Model  string `json:"model,omitempty"`
	Size   string `json:"size,omitempty"`
	Device string `json:"device,omitempty"`
	Type   string `json:"type,omitempty"`
	DiskID string `json:"diskID,omitempty"`
	PartID string `json:"partID,omitempty"`
}

var (
	devices []*Device
)

func init() {
	devices = scanDevices()
}

func GetDevice(diskId, partId string) string {
	for i := 0; i < len(devices); i++ {
		if strings.ToUpper(devices[i].DiskID) == strings.ToUpper(diskId) && strings.ToUpper(devices[i].PartID) == strings.ToUpper(partId) {
			return devices[i].Device
		}
	}
	return ""
}

func scanDevices() []*Device {
	d := make([]*Device, 0)

	// Use the "lsblk" command to list all block devices
	out, err := exec.Command("lsblk", "-o", "NAME,PTTYPE,PTUUID,PARTUUID,TYPE,SIZE,MODEL", "-P").Output()
	if err != nil {
		fmt.Println("Error executing lsblk:", err)
		return d
	}

	// Split the output into lines
	lines := strings.Split(string(out), "\n")

	// Loop through each line
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Split the line into key-value pairs
		pairs := strings.Split(line, "\"")
		if len(pairs) < 6 {
			continue
		}

		devname := pairs[1]
		devtype := pairs[9]
		devsize := pairs[11]
		devcode := pairs[13]

		var gpt bool
		var diskid string
		var partid string

		if pairs[3] == "gpt" {
			gpt = true
			diskid = pairs[5]
			partid = pairs[7]
		} else {
			gpt = false
			diskid = strings.Replace(pairs[5], "-", "", -1)
			diskidInt, _ := strconv.ParseInt(diskid, 16, 64)
			diskid = fmt.Sprintf("%d", diskidInt)

			partid = pairs[7]
			partidInt, _ := strconv.ParseInt(partid, 16, 64)
			partid = fmt.Sprintf("%d", partidInt*512)
		}

		d = append(d, &Device{gpt, devcode, devsize, devname, devtype, strings.ToUpper(diskid), strings.ToUpper(partid)})
	}

	return d
}
