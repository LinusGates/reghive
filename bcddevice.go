package reghive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/LinusGates/osmgr"
)

func GuidFrom(b []byte) string {
	t1 := binary.LittleEndian.Uint32(b[:4])
	t2 := binary.LittleEndian.Uint16(b[4:6])
	t3 := binary.LittleEndian.Uint16(b[6:8])
	t4 := binary.BigEndian.Uint16(b[8:10])
	t5 := binary.BigEndian.Uint32(b[10:14])
	t6 := binary.BigEndian.Uint16(b[14:16])

	l := make([]string, 6)
	l[0] = fmt.Sprintf("%08s", strconv.FormatUint(uint64(t1), 16))
	l[1] = fmt.Sprintf("%04s", strconv.FormatUint(uint64(t2), 16))
	l[2] = fmt.Sprintf("%04s", strconv.FormatUint(uint64(t3), 16))
	l[3] = fmt.Sprintf("%04s", strconv.FormatUint(uint64(t4), 16))
	l[4] = fmt.Sprintf("%08s", strconv.FormatUint(uint64(t5), 16))
	l[5] = fmt.Sprintf("%04s", strconv.FormatUint(uint64(t6), 16))

	return strings.ToUpper(l[0] + "-" + l[1] + "-" + l[2] + "-" + l[3] + "-" + l[4] + l[5])
}

func DeviceEntryFrom(b []byte) ([]byte, string) {
	var guid string
	for _, v := range b[:0x10] {
		if v != 0 {
			guid = GuidFrom(b[:0x10])
			break
		}
	}
	return b[0x10:], guid
}

func PacketFrom(b []byte) ([]byte, uint32, uint32, uint32, uint32, []byte) {
	header := []uint32{binary.LittleEndian.Uint32(b[0:4]), binary.LittleEndian.Uint32(b[4:8]), binary.LittleEndian.Uint32(b[8:12]), binary.LittleEndian.Uint32(b[12:16])}
	return b[header[2]:], header[0], header[1], header[2], header[3], b[0x10:header[2]]
}

func DiskPartitionFrom(b []byte) ([]byte, string, bool, string) {
	partid := b[:0x10]
	u3 := binary.LittleEndian.Uint32(b[0x10:0x14])
	tabletype := binary.LittleEndian.Uint32(b[0x14:0x18])
	diskid := b[0x18:0x28]
	u4 := []uint32{binary.LittleEndian.Uint32(b[0x28:0x2c]), binary.LittleEndian.Uint32(b[0x2c:0x30]), binary.LittleEndian.Uint32(b[0x30:0x34]), binary.LittleEndian.Uint32(b[0x34:0x38])}
	if u3 != 0 || u4[0] != 0 || u4[1] != 0 || u4[2] != 0 || u4[3] != 0 {
		return nil, "", false, "" //unexpected value
	}
	var partidValue string
	var diskidValue string
	if tabletype == 0 {
		partidValue = GuidFrom(partid)
		diskidValue = GuidFrom(diskid)
		return b[0x38:], partidValue, true, diskidValue
	} else if tabletype == 1 {
		partidValue = strings.ToUpper(fmt.Sprintf("%d", binary.LittleEndian.Uint64(partid)))
		diskidValue = strings.ToUpper(fmt.Sprintf("%d", binary.LittleEndian.Uint64(diskid)))
		return b[0x38:], partidValue, false, diskidValue
	}
	return nil, "", false, "" //unknown disk/partition ID
}

func DiskFileFrom(b []byte) ([]byte, uint32, uint32, []byte, string) {
	dtype, _, _, _ := binary.LittleEndian.Uint32(b[:4]), binary.LittleEndian.Uint32(b[4:8]), binary.LittleEndian.Uint32(b[8:12]), binary.LittleEndian.Uint32(b[12:16])
	b, ptype, _, _, _, data := PacketFrom(b[0x10:])
	pos := bytes.Index(b, []byte{0x00, 0x00, 0x00})
	pathB := b[:pos+1]
	path16 := make([]uint16, len(pathB)/2)
	reader := bytes.NewReader(pathB)
	binary.Read(reader, binary.LittleEndian, &path16)
	path := string(utf16.Decode(path16))
	return b[pos+3:], dtype, ptype, data, path
}

func RamDiskFrom(b []byte) ([]byte, [9]uint32, uint32, []byte, string) {
	var u9 [9]uint32
	reader := bytes.NewReader(b[:0x24])
	err := binary.Read(reader, binary.LittleEndian, &u9)
	if err != nil {
		log.Fatalf("binary.Read failed: %v", err)
	}
	b = b[0x24:]
	if u9[0] != 3 { //|| any(u9[1:5]) {
		log.Fatalf("Unexpected values in u9")
	}
	b, ptype, _, _, _, data := PacketFrom(b)
	pos := bytes.Index(b, []byte{0x00, 0x00, 0x00})
	pathB := b[:pos+1]
	path16 := make([]uint16, len(pathB)/2)
	reader = bytes.NewReader(pathB)
	binary.Read(reader, binary.LittleEndian, &path16)
	path := string(utf16.Decode(path16))
	return b[pos+3:], u9, ptype, data, path
}

func VhdDiskFrom(b []byte) ([]byte, uint32, uint32, []byte) {
	_, locatecustom, _, _ := binary.LittleEndian.Uint32(b[:4]), binary.LittleEndian.Uint32(b[4:8]), binary.LittleEndian.Uint32(b[8:12]), binary.LittleEndian.Uint16(b[12:14])
	b, ptype, _, _, _, data := PacketFrom(b[0x0E:])
	return b, locatecustom, ptype, data
}

func VhdDiskFileFrom(b []byte) ([]byte, uint32, []byte) {
	_ = binary.LittleEndian.Uint32(b[:4])
	b, ptype, _, _, _, data := PacketFrom(b[0x18:])
	return b, ptype, data
}

type BCDDevice struct {
	GPT          bool   `json:"gpt,omitempty"`
	Type         string `json:"type,omitempty"`
	Disk         string `json:"disk,omitempty"`
	Device       string `json:"device,omitempty"`
	DiskID       string `json:"diskID,omitempty"`
	PartID       string `json:"partID,omitempty"`
	GUID         string `json:"guid,omitempty"`
	Path         string `json:"path,omitempty"`
	LocateCustom uint32 `json:"locateCustom,omitempty"`
}

func (dev *BCDDevice) String() string {
	str := ""
	if dev.Device != "" {
		str += "/dev/" + dev.Device + ":"
	} else {
		str += "MISSING:"
	}
	if dev.GPT {
		str += "GPT"
	} else {
		str += "MBR"
	}
	if dev.Type == dev.Disk {
		str += " Type:" + dev.Type
	} else {
		str += " Type:" + dev.Type + " DiskType:" + dev.Disk
	}
	if dev.DiskID != "" {
		str += " Disk:" + dev.DiskID
	}
	if dev.PartID != "" {
		str += " Partition:" + dev.PartID
	}
	if dev.GUID != "" {
		str += " GUID:" + dev.GUID
	}
	if dev.Path != "" {
		str += " Path:" + dev.Path
	}
	if dev.LocateCustom != 0 {
		str += fmt.Sprintf(" LocateCustom:%d", dev.LocateCustom)
	}
	return str
}

func BCDDeviceFromBin(b []byte) (dev *BCDDevice, err error) {
	dev = &BCDDevice{}

	b, dev.GUID = DeviceEntryFrom(b)
	_, ptype, u1, _, _, b := PacketFrom(b)

	switch ptype {
	case 0: //file
		if u1 == 0 {
			dev.Type = "file"
			_, _, ptype, b, dev.Path = DiskFileFrom(b)
			if ptype == 5 {
				dev.Disk = "boot"
			} else {
				dev.Disk = "partition"
				b, dev.PartID, dev.GPT, dev.DiskID = DiskPartitionFrom(b)
			}
		} else {
			dev.Type = "ramdisk"
			_, _, ptype, b, dev.Path = RamDiskFrom(b)
			if ptype == 5 {
				dev.Disk = "boot"
			} else {
				dev.Disk = "partition"
				b, dev.PartID, dev.GPT, dev.DiskID = DiskPartitionFrom(b)
			}
		}
	case 5: //boot
		dev.Type = "boot"
		dev.Disk = "boot"
		return
	case 6: //partition
		dev.Type = "partition"
		dev.Disk = "partition"
		b, dev.PartID, dev.GPT, dev.DiskID = DiskPartitionFrom(b)
	case 8: //vhd/locate
		dev.Type = "vhd"
		var ptype uint32
		_, dev.LocateCustom, ptype, b = VhdDiskFrom(b)
		_, ptype, b = VhdDiskFileFrom(b)
		_, _, ptype, b, dev.Path = DiskFileFrom(b)
		if ptype == 5 {
			dev.Disk = "boot"
		} else if ptype == 8 {
			dev.Disk = "locate"
		} else {
			dev.Disk = "partition"
			b, dev.PartID, dev.GPT, dev.DiskID = DiskPartitionFrom(b)
		}
	default:
		return nil, fmt.Errorf("Unknown packet type: %d", ptype)
	}

	if dev.Type == "partition" {
		disk := osmgr.GetDisk(dev.DiskID)
		if disk != nil {
			part := disk.GetPartition(dev.PartID)
			if part != nil {
				dev.Device = part.Block
			} else {
				dev.Device = disk.Block
			}
		}
	}

	return dev, nil
}
