package reghive

import (
	"crypto/rand"
	"fmt"
	"strings"

	hivex "github.com/gabriel-samfira/go-hivex"
)

type Registry struct {
	Hive   *HiveNode `json:"hive"`
	closed bool      `json:"-"`
}

func OpenRegistryHive(path string) (*Registry, error) {
	hive, err := hivex.NewHivex(path, hivex.READ|hivex.WRITE|hivex.UNSAFE|hivex.VERBOSE)
	if err != nil {
		return nil, err
	}
	root, err := hive.Root()
	if err != nil {
		return nil, err
	}
	rootNode, err := decodeHiveNode(hive, root)
	if err != nil {
		return nil, err
	}
	return &Registry{Hive: rootNode}, nil
}

func (registry *Registry) GetNode(path string) (*HiveNode, error) {
	path = strings.ReplaceAll(path, "\\", "/")
	if path == "" || path == "/" {
		return registry.Hive, nil
	}
	//Node keys are case insensitive, and so will be this search
	path = strings.ToLower(path)
	//Split the requested path into each node key to nest into
	nodes := strings.Split(path, "/")
	//Strip path node shenanigans
	if nodes[0] == "" { //path starts with /
		nodes = nodes[1:]
	}
	if nodes[len(nodes)-1] == "" { //path ends with /
		nodes = nodes[:len(nodes)-1]
	}

	//Loop over each key in the path, nesting deeper into the node structure each time
	retNode := registry.Hive
	for i := 0; i < len(nodes); i++ {
		//Loop over each node in the current return node to check for the next key from the path
		var testNode *HiveNode
		for _, childNode := range retNode.Children {
			//Check if the raw key or the known node key matches the current node key
			if strings.ToLower(childNode.Name) == nodes[i] || NodeKeyVal(childNode.Name) == nodes[i] {
				//Set the node to use and break from the loop
				testNode = childNode
				break
			}
		}
		if testNode == nil {
			return nil, fmt.Errorf("GetNode: Could not find path node %s", nodes[i])
		}
		//Set the node to return in case this was the last node to nest into
		retNode = testNode
	}

	return retNode, nil
}

// Close frees the hivex reference, making this registry safe to free, but discards any unsaved changes
func (registry *Registry) Close() error {
	if registry.closed {
		return fmt.Errorf("registry: Already closed")
	}
	if err := registry.Hive.hive.Close(); err != nil {
		return err
	}
	registry.closed = true
	return nil
}

type HiveNode struct {
	Name     string       `json:"name"`
	Children []*HiveNode  `json:"nodes,omitempty"`
	Values   []*HiveValue `json:"values,omitempty"`

	//Internal usage for an open registry hive
	hive     *hivex.Hivex `json:"-"`
	hiveNode int64        `json:"-"`
}

func (n *HiveNode) String() string {
	return TreeNode(n, 0)
}

func TreeNode(node *HiveNode, indent int) string {
	indentChar := "\t"
	indentLine := ""
	for i := 0; i < indent; i++ {
		indentLine += indentChar
	}
	tree := fmt.Sprintf("%sNode: %s = %s\n", indentLine, node.Name, NodeKeyVal(node.Name))

	if len(node.Values) > 0 {
		tree += fmt.Sprintf("%s%sValues: %d\n", indentLine, indentChar, len(node.Values))
		for i := 0; i < len(node.Values); i++ {
			tree += fmt.Sprintf("%s%s%s = %s == %s\n", indentLine, indentChar, node.Values[i].Key, NodeKeyVal(node.Values[i].Key), node.Values[i].String())
		}
	}

	if len(node.Children) > 0 {
		tree += fmt.Sprintf("%s%sChildren: %d\n", indentLine, indentChar, len(node.Children))
		for i := 0; i < len(node.Children); i++ {
			tree += TreeNode(node.Children[i], indent+1)
		}
	}

	return tree
}

func decodeHiveNode(hive *hivex.Hivex, node int64) (*HiveNode, error) {
	nodeName, err := hive.NodeName(node)
	if err != nil {
		return nil, err
	}
	//nodeName = NodeKeyVal(nodeName)
	nodeChildren, err := hive.NodeChildren(node)
	if err != nil {
		return nil, err
	}
	nodeValues, err := hive.NodeValues(node)
	if err != nil {
		return nil, err
	}

	children := make([]*HiveNode, len(nodeChildren))
	for i := 0; i < len(nodeChildren); i++ {
		nodeChild, err := decodeHiveNode(hive, nodeChildren[i])
		if err != nil {
			return nil, err
		}
		children[i] = nodeChild
	}

	values := make([]*HiveValue, len(nodeValues))
	for i := 0; i < len(nodeValues); i++ {
		nodeValue, err := decodeHiveValue(hive, node, nodeValues[i])
		if err != nil {
			return nil, err
		}
		values[i] = nodeValue
	}

	return &HiveNode{Name: nodeName, Children: children, Values: values, hive: hive, hiveNode: node}, nil
}

type HiveValue struct {
	Key           string       `json:"key,omitempty"`
	Value         int64        `json:"value,omitempty"`
	ValueType     RegValueType `json:"type"`
	ValueBytes    []byte       `json:"bytes,omitempty"`
	ValueString   string       `json:"string,omitempty"`
	ValueStrings  []string     `json:"strings,omitempty"`
	ValueDevice   *BCDDevice   `json:"device,omitempty"`
	ValueDescType *BCDDescType `json:"bcdflags,omitempty"`

	//Internal usage for an open registry hive
	hive     *hivex.Hivex `json:"-"`
	hiveNode int64        `json:"-"`
}

func (v *HiveValue) String() string {
	switch v.ValueType {
	case RegSZ:
		return NodeKeyVal(v.ValueString)
	case RegMultiSZ:
		valStrings := v.ValueStrings
		for i := 0; i < len(valStrings); i++ {
			valStrings[i] = NodeKeyVal(valStrings[i])
		}
		return strings.Join(valStrings, ", ")
	case RegDevice:
		if v.ValueDevice != nil {
			return v.ValueDevice.String()
		}
	case RegDescType:
		if v.ValueDescType != nil {
			valueData := v.ValueDescType
			return fmt.Sprintf("%X == %s", valueData.Source, valueData.String())
		}
	}

	valueData := v.ValueBytes
	return fmt.Sprintf("%s == 0x%X", valueData, valueData)
}

func decodeHiveValue(hive *hivex.Hivex, node, value int64) (*HiveValue, error) {
	valueKey, err := hive.NodeValueKey(value)
	if err != nil {
		return nil, err
	}
	//valueKey = NodeKeyVal(valueKey)
	valueType, valueBytes, err := hive.ValueValue(value)
	if err != nil {
		return nil, err
	}
	parentNodeName, err := hive.NodeName(node)
	if err == nil {
		//parentNodeName = NodeKeyVal(parentNodeName)
	}

	hiveValue := &HiveValue{Key: valueKey, Value: value, ValueType: RegValueType(valueType), ValueBytes: valueBytes, hive: hive, hiveNode: node}

	switch hiveValue.ValueType {
	case RegBinary:
		switch parentNodeName {
		case "device", "osdevice", "ramdisksdidevice":
			bcdDevice, err := BCDDeviceFromBin(valueBytes)
			if err != nil {
				return nil, err
			}
			hiveValue.ValueType = RegDevice
			hiveValue.ValueDevice = bcdDevice
			hiveValue.ValueBytes = nil
		}
	case RegDwordLittle:
		if parentNodeName == "Description" && valueKey == "Type" {
			bcdDescType := NewBCDDescType(valueBytes)
			if bcdDescType != nil {
				hiveValue.ValueType = RegDescType
				hiveValue.ValueDescType = bcdDescType
				hiveValue.ValueBytes = nil
			}
		}
	case RegSZ, RegExpandSZ:
		value, err := DecodeUTF16(valueBytes)
		if err != nil {
			value = string(valueBytes)
		}
		//value = NodeKeyVal(value)
		hiveValue.ValueString = value
		hiveValue.ValueBytes = nil
	case RegMultiSZ:
		values, err := hive.ValueMultipleStrings(value)
		if err != nil {
			return nil, err
		}
		values = values[:len(values)-1]
		for i := 0; i < len(values); i++ {
			//values[i] = NodeKeyVal(values[i])
		}
		hiveValue.ValueStrings = values
		hiveValue.ValueBytes = nil
	}

	return hiveValue, nil
}

func GenerateGuid() (string, error) {
	guidBuf := make([]byte, 16)
	if _, err := rand.Read(guidBuf); err != nil {
		return "", err
	}
	return GuidFrom(guidBuf), nil
}
