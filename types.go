package reghive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	maskObject      = 0xF0000000
	maskImage       = 0x00F00000
	maskApplication = 0x000FFFFF
)

type ObjectType int

func (t ObjectType) String() string {
	switch t {
	case ObjectApplication:
		return "objectApplication"
	case ObjectInherit:
		return "objectInherit"
	case ObjectDevice:
		return "objectDevice"
	}
	return "objectNULL"
}

const (
	ObjectApplication ObjectType = iota + 1
	ObjectInherit
	ObjectDevice
)

type ImageType int

func (t ImageType) String() string {
	switch t {
	case ImageFirmware:
		return "imageFirmware"
	case ImageWindowsBoot:
		return "imageWindowsBoot"
	case ImageLegacyLoader:
		return "imageLegacyLoader"
	case ImageRealMode:
		return "imageRealMode"
	}
	return "imageNULL"
}

const (
	ImageFirmware ImageType = iota + 1
	ImageWindowsBoot
	ImageLegacyLoader
	ImageRealMode
)

type InheritType int

func (t InheritType) String() string {
	switch t {
	case InheritAnyObject:
		return "inheritAnyObject"
	case InheritApplicationObject:
		return "inheritApplicationObject"
	case InheritDeviceObject:
		return "inheritDeviceObject"
	}
	return "inheritNULL"
}

const (
	InheritAnyObject InheritType = iota + 1
	InheritApplicationObject
	InheritDeviceObject
)

type ApplicationType int

func (t ApplicationType) String() string {
	switch t {
	case FWBootmgr:
		return "fwbootmgr"
	case Bootmgr:
		return "bootmgr"
	case OsLoader:
		return "osloader"
	case Resume:
		return "resume"
	case MemDiag:
		return "memdiag"
	case Ntldr:
		return "ntldr"
	case Setupldr:
		return "setupldr"
	case BootSector:
		return "bootsector"
	case Startup:
		return "startup"
	case BootApp:
		return "bootapp"
	}
	return "applicationNULL"
}

const (
	FWBootmgr ApplicationType = iota + 1
	Bootmgr
	OsLoader
	Resume
	MemDiag
	Ntldr
	Setupldr
	BootSector
	Startup
	BootApp
)

type BCDDescType struct {
	Source          []byte          `json:"-"`
	ObjectType      ObjectType      `json:"object"`
	ImageType       ImageType       `json:"image"`
	InheritType     InheritType     `json:"inherit"`
	ApplicationType ApplicationType `json:"application"`
}

func (desc *BCDDescType) String() string {
	return fmt.Sprintf("%s, %s, %s, %s", desc.ObjectType.String(), desc.ImageType.String(), desc.InheritType.String(), desc.ApplicationType.String())
}

func NewBCDDescType(descType []byte) *BCDDescType {
	var typeDWORD uint32
	binary.Read(bytes.NewReader(descType), binary.LittleEndian, &typeDWORD)

	objectType := ObjectType((typeDWORD & maskObject) >> 28)
	imageType := ImageType((typeDWORD & maskImage) >> 20)
	inheritType := InheritType((typeDWORD & maskImage) >> 20)
	applicationType := ApplicationType(typeDWORD & maskApplication)

	return &BCDDescType{Source: descType, ObjectType: objectType, ImageType: imageType, InheritType: inheritType, ApplicationType: applicationType}
}

type RegValueType int64

const (
	RegNone RegValueType = iota
	RegSZ
	RegExpandSZ
	RegBinary
	RegDwordLittle
	RegDwordBig
	RegLink
	RegMultiSZ
	RegResourceList
	RegQwordLittle = 0xB
	RegDevice      = 0xE
	RegDescType    = 0xF
)

func (t RegValueType) String() string {
	switch t {
	case RegNone:
		return "REG_NONE"
	case RegSZ:
		return "REG_SZ"
	case RegExpandSZ:
		return "REG_EXPAND_SZ"
	case RegBinary:
		return "REG_BINARY"
	case RegDwordLittle:
		return "REG_DWORD_LITTLE"
	case RegDwordBig:
		return "REG_DWORD_BIG"
	case RegLink:
		return "REG_LINK"
	case RegMultiSZ:
		return "REG_MULTI_SZ"
	case RegResourceList:
		return "REG_RESOURCE_LIST"
	case RegQwordLittle:
		return "REG_QWORD_LITTLE"
	case RegDevice:
		return "REG_BCD_DEVICE"
	case RegDescType:
		return "REG_BCD_DESCTYPE"
	}
	return "REG_NULL"
}

func DecodeUTF16(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", fmt.Errorf("Must have even length byte slice")
	}

	for i := len(b) - 2; i >= 0; i -= 2 {
		if b[i] == 0x0 && b[i+1] == 0x0 {
			b = b[:i]
		}
	}
	if len(b) == 0 {
		return "", nil
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}
