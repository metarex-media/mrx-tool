package mrxUnitTest

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"

	"golang.org/x/sync/errgroup"
)

// Node is a object in the abstact syntax tree
// it can be a child, a parent or both.
type Node struct {
	Key, Length, Value Position
	Properties         MXFProperty
	// talk through the children role with Bruce
	// but keep as this
	Tests    tests[Node]
	Children []*Node
}

type Nodes interface {
	Node | PartitionNode | MXFNode
}

type parent interface {
	callBack() // a function that signal infected child
}

func (n *Node) callBack() {
	n.Tests.TestPass = false
	n.Tests.parent.callBack()

}

func (p *PartitionNode) callBack() {
	p.Tests.TestPass = false
	p.Tests.parent.callBack()
	//	.callBack()
}

func (m *MXFNode) callBack() {
	m.Tests.TestPass = false
}

type tests[N Nodes] struct {
	parent          parent `yaml:"-"`
	tests           []*func(doc io.ReadSeeker, header *N) func(t Test)
	testsWithPrimer []*func(doc io.ReadSeeker, header *N, primer map[string]string) func(t Test)
	TestPass        bool
}

type MXFNode struct {
	Partitions []*PartitionNode
	Tests      tests[MXFNode]
}

type PartitionNode struct {
	Parent             *MXFNode `yaml:"-"`
	Key, Length, Value Position
	HeaderMetadata     []*Node
	Essence            []*Node
	IndexTable         *Node
	Props              PartitionProperties
	Tests              tests[PartitionNode]
	PartitionPos       int
}

// FindUL returns the first Node with that symbol found in the
// Node Tree. Depth first search
func (n *Node) FindUL(sym string) *Node {
	if n == nil {
		return nil
	}
	for _, n := range n.Children {

		if n != nil {
			if n.Properties != nil {
				if n.Properties.UL() == sym {
					return n
				}

				// check the childrens children
				found := n.FindUL(sym)
				if found != nil {
					return found
				}
			}
		}

	}
	return nil
}

// FindSymbol returns all the Nodes with the universal label(s) found in the
// Node Tree.
func (n *Node) FindULs(sym ...string) []*Node {

	if n == nil {
		return nil
	}

	foundNodes := make([]*Node, 0)

	for _, n := range n.Children {

		if n != nil {
			if n.Properties != nil {
				if slices.Contains(sym, n.Properties.UL()) {
					foundNodes = append(foundNodes, n)
				}

				// check the childrens children
				found := n.FindULs(sym...)
				if found != nil {
					foundNodes = append(foundNodes, found...)
				}
			}
		}

	}
	if len(foundNodes) > 0 {
		return foundNodes
	}

	return nil
}

// FindSymbol returns all the Nodes with the symbol(s) found in the
// Node Tree.
func (n *Node) FindTypes(typ ...string) []*Node {

	if n == nil {
		return nil
	}

	foundNodes := make([]*Node, 0)

	for _, n := range n.Children {

		if n != nil {
			if n.Properties != nil {
				for _, label := range n.Properties.Label() {
					if slices.Contains(typ, label) {
						foundNodes = append(foundNodes, n)
					}
				}

				// check the childrens children
				found := n.FindTypes(typ...)
				if found != nil {
					foundNodes = append(foundNodes, found...)
				}
			}
		}

	}
	if len(foundNodes) > 0 {
		return foundNodes
	}

	return nil
}

// Position is a demo position for this library
// @TODO update it
type Position struct {
	Start, End int
}

type MXFProperty interface {
	// symbol returns the MXF UL associated with the node.
	// if there is one
	UL() string
	//ID returns the ID associated with the property
	ID() string
	// Returns the type of that node
	// e.g. essence, partition or the group type like Descriptivemetadata
	Label() []string
}

type EssenceProperties struct {
	EssUL string
}

func (e EssenceProperties) ID() string {

	return ""
}

const EssenceLabel = "essence"

func (e EssenceProperties) Label() []string {

	return []string{EssenceLabel}
}

// symbol returns the partition type
func (e EssenceProperties) UL() string {
	return e.EssUL
}

type GroupProperties struct {
	UUID           mxf2go.TUUID
	UniversalLabel string
	GroupLabel     []string
}

func (gp GroupProperties) ID() string {
	var fullUUID string
	for _, uid := range gp.UUID {
		fullUUID += fmt.Sprintf("%02x", uid)
	}
	return fullUUID
}

func (gp GroupProperties) UL() string {
	return gp.UniversalLabel
}

func (gp GroupProperties) Label() []string {
	return gp.GroupLabel
}

type PartitionProperties struct {
	PartitionCount int // the count of the partition along the MXF
	PartitionType  string
	Primer         map[string]string
	EssenceOrder   []string
}

func (p PartitionProperties) ID() string {

	return ""
}

const PartitionType = "partition"

func (p PartitionProperties) Label() []string {

	return []string{PartitionType}
}

// symbol returns the partition type
func (p PartitionProperties) Symbol() string {
	//fmt.Println(p.PartitionType)
	return p.PartitionType
}

// Search follows SQL for finding things within a partition
// e.g. select * from essence where UL <> 060e2b34.01020105.0e090502.017f017f
//
// The search command is not case sensitive
func (p PartitionNode) Search(searchfield string) ([]*Node, error) {
	//lowercase as ULs are lower case when searching
	command := strings.Split(strings.ToLower(searchfield), " ")

	if len(command) < 4 {
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field\" as a minimum command", searchfield)
	}

	if command[0] != "select" {
		return nil, fmt.Errorf("first word not select")
	}

	// worry about this later
	// if command[1] != "*"

	var searchFields []*Node
	switch command[3] {
	case "essence":
		searchFields = p.Essence
	case "metadata":
		searchFields = p.HeaderMetadata
	default:
		return nil, fmt.Errorf("invalid field of \"%s\"", command[3])
	}

	switch len(command) {
	case 4:
		return searchFields, nil
	case 8:
		// keep on trucking
	default:
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field where x = y\" as a minimum command", searchfield)
	}

	out := make([]*Node, 0)
	for _, search := range searchFields {
		founds, err := recurseSearch(search, command[5], command[6], command[7])
		if err != nil {
			return nil, err
		}
		// search through the children as well
		out = append(out, founds...)
	}
	return out, nil
}

func recurseSearch(node *Node, field, equate, target string) ([]*Node, error) {

	if node == nil {
		return nil, nil
	}
	out := make([]*Node, 0)

	// search through the children as well
	var compareField string

	switch field {
	case "ul":

		compareField = node.Properties.UL()
	default:
		return nil, fmt.Errorf("unknown field \"%v\"", field)
	}

	var pass bool
	switch equate {
	case "=":
		pass = (compareField == target)
	case "<>":
		pass = (compareField != target)
	default:
		return nil, fmt.Errorf("unknown comparison operator \"%v\"", equate)
	}

	if pass {
		out = append(out, node)
	}

	for _, child := range node.Children {
		founds, err := recurseSearch(child, field, equate, target)
		if err != nil {
			return nil, err
		}
		// search through the children as well
		out = append(out, founds...)
	}

	return out, nil
}

// Search follows SQL for finding things within a partition
// e.g. select * from essence where UL <> 060e2b34.01020105.0e090502.017f017f
//
// The search command is not case sensitive
func (m MXFNode) Search(searchfield string) ([]*PartitionNode, error) {
	//lowercase as ULs are lower case when searching
	command := strings.Split(strings.ToLower(searchfield), " ")

	if len(command) < 4 {
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field\" as a minimum command", searchfield)
	}

	if command[0] != "select" {
		return nil, fmt.Errorf("first word not select")
	}

	// worry about this later
	// if command[1] != "*"

	var searchFields []*PartitionNode
	switch command[3] {
	case "partition", "partitions":
		searchFields = m.Partitions
	default:
		return nil, fmt.Errorf("invalid field of \"%s\"", command[3])
	}

	switch len(command) {
	case 4:
		return searchFields, nil
	case 8:
		// keep on trucking
	default:
		return nil, fmt.Errorf("malformed command of %s expected \"select * from field where x = y\" as a minimum command", searchfield)
	}

	out := make([]*PartitionNode, 0)
	for _, search := range searchFields {
		var compareField string

		switch command[5] {
		case "type":
			compareField = search.Props.PartitionType
		default:
			return nil, fmt.Errorf("unknown field \"%v\"", command[5])
		}

		var pass bool
		switch command[6] {
		case "=":
			pass = (compareField == command[7])
		case "<>":
			pass = (compareField != command[7])
		default:
			return nil, fmt.Errorf("unknown comparison operator \"%v\"", command[6])

		}

		if pass {
			out = append(out, search)
		}
	}
	return out, nil
}

/*
have to be more explicit than the go AST
implement types of node?

Parition Node
Group Node
Essence Node - these can all be taken away with the properties
			 - handle properties of type group property, essence etc. Can still be filtered out as
			   yaml, but handled anyonmously inside. Allows people to put their own properties in


*/

// Add a search feature in
// search by property? Properties map[comparable]any? Less type assertion more searching
// would omit properties?

/*

search the map as well based on UUID?

If i want to find types of group X or only essence.

Everything has a depth so when its printed?


Wnat to replicate that nested view of  XML reg - which is where the anys come in
// do not parse everything, especially for large objects.

*/

/*
workflow

Include a control Node - these can be anything, include a search function and the Node as a list?
This contains detail shared across the file for MRX files

new parsing, each partition is a parent

*/

type refAndChild struct {
	child bool
	ref   [][]byte
}

type Specifications struct {
	// node specifications for groups, map is UL node test
	Node map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test)
	// test aprtitions the partition tyoe is the map key
	Part map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test)
	// array of mxf structual tests
	MXF  []*func(doc io.ReadSeeker, isxdDesc *MXFNode) func(t Test)
}

// inlcude the logger? if there's any errors flush them - discard ifo for unkown keys fro the moment
func MakeAST(stream io.Reader, buffer chan *klv.KLV, size int, specs Specifications) (*MXFNode, error) { // wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, size)
	})

	mxf := &MXFNode{Partitions: make([]*PartitionNode, 0), Tests: tests[MXFNode]{TestPass: true, tests: specs.MXF}}
	var currentPartitionNode *PartitionNode
	// /	var currentPartition int
	var primer map[string]string
	// @TODO set this up with errs so test breaking errors are returned
	errs.Go(func() error {

		defer func() {
			// this only runs when an error occurs to stop blocking
			_, klvOpen := <-buffer
			for klvOpen {
				_, klvOpen = <-buffer
			}

		}()

		// get the first bit of stream
		klvItem, klvOpen := <-buffer

		offset := 0
		var patternTally bool
		// handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence
			if partitionName(klvItem.Key) == "060e2b34.020501  .0d010201.01    00" {
				if currentPartitionNode != nil {
					mxf.Partitions = append(mxf.Partitions, currentPartitionNode)
				}

				// add details from now
				currentPartitionNode = &PartitionNode{

					Key:            Position{Start: offset, End: offset + len(klvItem.Key)},
					Length:         Position{Start: offset + len(klvItem.Key), End: offset + len(klvItem.Key) + len(klvItem.Length)},
					Value:          Position{Start: offset + len(klvItem.Key) + len(klvItem.Length), End: offset + klvItem.TotalLength()},
					HeaderMetadata: make([]*Node, 0),
					Essence:        make([]*Node, 0),
					Parent:         mxf,
					Tests:          tests[PartitionNode]{TestPass: true, parent: mxf},
					PartitionPos:   len(mxf.Partitions),
				}
				patternTally = true

				// create a reference map for every node that is found
				refMap := make(map[*Node]refAndChild)
				offset += klvItem.TotalLength()
				// test the previous partitions essence as the final step
				// if len(contents.RipLayout) == 0 and the cache length !=0 emit an error that essence was found first

				partProps := PartitionProperties{PartitionCount: len(mxf.Partitions), EssenceOrder: make([]string, 0)}

				switch klvItem.Key[13] {
				case 17:
					partProps.PartitionType = RIPPartition
				case 02:
					// header
					partProps.PartitionType = HeaderPartition
					currentPartitionNode.Tests.tests = append(currentPartitionNode.Tests.tests, specs.Part[HeaderKey]...)
				case 03:
					// body
					if klvItem.Key[14] == 17 {
						partProps.PartitionType = GenericStreamPartition
						currentPartitionNode.Tests.tests = append(currentPartitionNode.Tests.tests, specs.Part[GenericKey]...)

					} else {
						partProps.PartitionType = BodyPartition
						currentPartitionNode.Tests.tests = append(currentPartitionNode.Tests.tests, specs.Part[EssenceKey]...)
					}
				case 04:
					// footer
					partProps.PartitionType = FooterPartition
					currentPartitionNode.Tests.tests = append(currentPartitionNode.Tests.tests, specs.Part[HeaderKey]...)

				default:
					// is nothing
					partProps.PartitionType = "invalid"

				}
				// primer will get updated because of pointer magic
				partProps.Primer = primer
				currentPartitionNode.Props = partProps

				partitionLayout := partitionExtract(klvItem)

				metaByteCount := 0
				idMap := make(map[string]*Node) // assign the ids of the map
				for metaByteCount < int(partitionLayout.HeaderByteCount) {
					flush, open := <-buffer

					if !open {
						return fmt.Errorf("error when using klv data klv stream interrupted")
					}
					// decode the essence here

					flushNode := &Node{
						Key:    Position{Start: offset, End: offset + len(flush.Key)},
						Length: Position{Start: offset + len(flush.Key), End: offset + len(flush.Key) + len(flush.Length)},
						Value:  Position{Start: offset + len(flush.Key) + len(flush.Length), End: offset + flush.TotalLength()},
						Tests:  tests[Node]{TestPass: true},
					}

					refMap[flushNode] = refAndChild{}

					// @TODO include KLV fill packets
					dec, skip := decodeBuilder(flush.Key[5])

					if skip {

						//unpack the primer

						if fullNameMask(flush.Key, 5) == "060e2b34.027f0101.0d010201.01050100" {
							out := make(map[string]string)
							primerUnpack(flush.Value, out)
							primer = out
							flushNode.Properties = GroupProperties{UniversalLabel: "060e2b34.027f0101.0d010201.01050100"}
							currentPartitionNode.Props.Primer = primer
						}
						// want to loop through them all?

					} else {

						decoders, ok := mxf2go.Groups["urn:smpte:ul:"+fullName(flush.Key)]

						if !ok {
							flush.Key[5] = 0x7f
							decoders, ok = mxf2go.Groups["urn:smpte:ul:"+fullName(flush.Key)]
						}
						if !ok {
							flush.Key[13] = 0x7f
							decoders, ok = mxf2go.Groups["urn:smpte:ul:"+fullName(flush.Key)]
						}

						// assign the generic name as the key
						key := fullName(flush.Key)
						flushNode.Properties = GroupProperties{UniversalLabel: key}
						// find the groups first

						if ok {
							if nodeTests, ok := specs.Node[key]; ok {

								flushNode.Tests = tests[Node]{testsWithPrimer: nodeTests}
							}
						}
						pos := 0
						for pos < len(flush.Value) {
							key, klength := dec.keyFunc(flush.Value[pos : pos+dec.keyLen])
							length, lenlength := dec.lengthFunc(flush.Value[pos+dec.keyLen : pos+dec.keyLen+dec.lengthLen])
							if klength != 16 {
								key = primer[key]
							}

							// @TODO inlude the key for other AUIDs and ObjectIDs as part of the process
							switch key {
							case "060e2b34.01010101.01011502.00000000":
								out, _ := mxf2go.DecodeTUUID(flush.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
								mid := flushNode.Properties.(GroupProperties)
								mid.UUID = out.(mxf2go.TUUID)
								flushNode.Properties = mid
								UUID := out.(mxf2go.TUUID)
								idMap[string(UUID[:])] = flushNode

							default:

								if ok {
									// check the decoder for the field
									decodeF, ok := decoders.Group["urn:smpte:ul:"+key]

									if ok {

										b, _ := decodeF.Decode(flush.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
										strongRefs := ReferenceExtract(b, strongRef)
										if len(strongRefs) > 0 {
											mid := refMap[flushNode]
											mid.ref = append(mid.ref, strongRefs...)
											refMap[flushNode] = mid
										} else {
											weakRefs := ReferenceExtract(b, weakRef)
											if len(weakRefs) != 0 {
												outString := make([]string, len(weakRefs))
												for i, wr := range weakRefs {
													outString[i] = fullName(wr)
												}

												mid := flushNode.Properties.(GroupProperties)
												mid.GroupLabel = outString
												flushNode.Properties = mid
											}
										}
									}
								}
							}
							pos += klength + length + lenlength
						}

						// "urn:smpte:ul:060e2b34.01010101.01011502.00000000"
					}

					offset += flush.TotalLength()
					metaByteCount += flush.TotalLength()

					// currentNode.Children = append(currentNode.Children, flushNode)

				}

				// thread the partition afterwards
				// first by finding the references
				// and marking if something is a child
				for n, refs := range refMap {
					for _, ref := range refs.ref {
						child := idMap[string(ref)]
						mid := refMap[child]
						mid.child = true
						refMap[child] = mid
						if child != nil {

							child.Tests.parent = n
						}
						n.Children = append(n.Children, child)
					}
				}

				// then by assigning all the parents
				for n, refs := range refMap {

					if !refs.child {
						n.Tests.parent = currentPartitionNode
						currentPartitionNode.HeaderMetadata = append(currentPartitionNode.HeaderMetadata, n)
					}
				}

				// order the map by appearance order
				slices.SortFunc(currentPartitionNode.HeaderMetadata, func(a, b *Node) int {
					return a.Key.Start - b.Key.Start
				})

				if partitionLayout.IndexTable {
					//	index table is after all the metadata
					index, open := <-buffer

					if !open {
						return fmt.Errorf("error parsing stream channel unexpectedly closed.")
					}
					currentPartitionNode.IndexTable = &Node{
						Key:    Position{Start: offset, End: offset + len(index.Key)},
						Length: Position{Start: offset + len(index.Key), End: offset + len(index.Key) + len(index.Length)},
						Value:  Position{Start: offset + len(index.Key) + len(index.Length), End: offset + index.TotalLength()},
						Tests:  tests[Node]{TestPass: true},
					}
					offset += index.TotalLength()

					//	fmt.Println(md.currentContainer.IndexTable)
				}

				//	currentPartitionNode.HeaderMetadata = append(currentPartitionNode.HeaderMetadata, currentPartitionNode)
			} else {
				// check the name as it came
				name := fullName(klvItem.Key)
				_, ok := mxf2go.EssenceLookUp["urn:smpte:ul:"+name]

				if len(currentPartitionNode.Props.EssenceOrder) != 0 {
					if currentPartitionNode.Props.EssenceOrder[0] == name {
						patternTally = false
					} else if patternTally {
						currentPartitionNode.Props.EssenceOrder = append(currentPartitionNode.Props.EssenceOrder, name)
					}
				} else {
					currentPartitionNode.Props.EssenceOrder = append(currentPartitionNode.Props.EssenceOrder, name)
				}

				if !ok {
					// check for a 7f masked version at the final byte
					klvItem.Key[15] = 0x7f
					_, ok = mxf2go.EssenceLookUp["urn:smpte:ul:"+fullName(klvItem.Key)]
					if !ok {
						// check for a 7f masked version at the final byte and the 14th byte
						klvItem.Key[13] = 0x7f
						_, ok = mxf2go.EssenceLookUp["urn:smpte:ul:"+fullName(klvItem.Key)]
						if ok {
							name = fullName(klvItem.Key)
						}
					} else {
						name = fullName(klvItem.Key)
					}
				}

				// the output symbol is the name of the key

				essNode := &Node{
					Key:        Position{Start: offset, End: offset + len(klvItem.Key)},
					Length:     Position{Start: offset + len(klvItem.Key), End: offset + len(klvItem.Key) + len(klvItem.Length)},
					Value:      Position{Start: offset + len(klvItem.Key) + len(klvItem.Length), End: offset + klvItem.TotalLength()},
					Properties: EssenceProperties{EssUL: name},
					Children:   make([]*Node, 0),
					Tests:      tests[Node]{TestPass: true, parent: currentPartitionNode},
				}

				currentPartitionNode.Essence = append(currentPartitionNode.Essence, essNode)
				offset += klvItem.TotalLength()
				// throw a warning here saying expected partition got KEY : fullname

			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}
		mxf.Partitions = append(mxf.Partitions, currentPartitionNode)
		return nil
	})

	// post processing data if the klv hasn't returned an error
	// count of partitions
	errs.Wait()

	//b, _ := yaml.Marshal(mxf)
	//dest.Write(b)
	//fmt.Println(mxf)
	// assign after the yaml to stop endless recursion
	for _, p := range mxf.Partitions {
		p.Parent = mxf
	}
	return mxf, nil
}

func primerUnpack(input []byte, shorthand map[string]string) {

	order := binary.BigEndian
	count := order.Uint32(input[0:4])
	length := order.Uint32(input[4:8]) // if length isn't 18 explode

	offset := 8
	for i := uint32(0); i < count; i++ {
		//fmt.Printf("%x: %v\n", input[offset:offset+2], fullName(input[offset+2:offset+18]))
		short := fmt.Sprintf("%04x", input[offset:offset+2])
		shorthand[short] = fullName(input[offset+2 : offset+18])
		offset += int(length)
	}

}

func oneNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 1 {
		return "", 0
	}

	return fmt.Sprintf("%02x", namebytes[0:1:1]), 1
}

/*
	func oneLengthKL(lengthbytes []byte) (int, int) {
		if len(lengthbytes) != 1 {
			return 0, 0
		}

		return int(lengthbytes[0]), 1
	}
*/
func twoNameKL(namebytes []byte) (string, int) {
	if len(namebytes) != 2 {
		return "", 0
	}

	return fmt.Sprintf("%04x", namebytes[0:2:2]), 2
}

func twoLengthKL(lengthbytes []byte) (int, int) {
	if len(lengthbytes) != 2 {
		return 0, 0
	}

	length := order.Uint16(lengthbytes[0:2:2])

	return int(length), 2
}

func fullNameKL(namebytes []byte) (string, int) {

	if len(namebytes) != 16 {
		return "", 0
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15]), 16
}

type keyLength struct {
	keyLen, lengthLen int
	lengthFunc        func([]byte) (int, int)
	keyFunc           func([]byte) (string, int)
}

// decodeBuilder generates the options to decode a packet.
// some tags need to be updated
func decodeBuilder(key uint8) (keyLength, bool) {
	var decodeOption keyLength
	var skip bool
	lenField := (key >> 4)
	keyField := (key & 0b00001111)

	// smpte 336 decode methods
	switch lenField {
	case 0, 1:
		decodeOption.lengthLen = 16
		decodeOption.lengthFunc = klv.BerDecode
	case 4, 5:
		decodeOption.lengthLen = 2
		decodeOption.lengthFunc = twoLengthKL
	default:
		skip = true
	}

	switch lenField%2 + keyField {
	case 0, 1, 2, 0xB:
		decodeOption.keyFunc = fullNameKL
		decodeOption.keyLen = 16
	case 4:
		decodeOption.keyFunc = twoNameKL
		decodeOption.keyLen = 2
	case 3:
		decodeOption.keyFunc = oneNameKL
		decodeOption.keyLen = 1
	case 0xC:
		// 3 is 1 byte
		// 0xB is ASN1
		// 0xC is 4
	default:
		skip = true
	}

	return decodeOption, skip
}

const (
	strongRef = "StrongReference"
	weakRef   = "WeakReference"
)

// map of UUID and parents
// if the uuid is found
// then assignt he child to the parents

// StrongReference checks if a type is strong reference,
// then recurisvely searches through the types to find the strong set version
func ReferenceExtract(field any, reftype string) [][]byte {

	switch v := field.(type) {
	case mxf2go.TStrongReference:
		return [][]byte{v}
	default:
		switch {
		case strings.Contains(reflect.TypeOf(field).Name(), reftype+"Set") || strings.Contains(reflect.TypeOf(field).Name(), reftype+"Vector"):
			arr := reflect.ValueOf(field)
			arrLen := arr.Len()
			referenced := make([][]byte, arrLen)

			for i := 0; i < arrLen; i++ {

				//id, _ := yaml.Marshal(arr.Index(i).Interface())

				arrField := arr.Index(i).Interface()
				//	_, ok := idmap[strid]
				// fmt.Println(strid, ok, []byte(strid))
				// the midmap ensures the preservation of the object order
				// result := StrongReference(idmap[string(id)].mapper, idmap)
				result := ReferenceExtract(arrField, reftype)
				referenced[i] = result[0]
			}

			return referenced
		case strings.Contains(reflect.TypeOf(field).Name(), reftype):
			return [][]byte{getId(v)}
		default:

			return [][]byte{}
		}
	}

}

// this just makes all the ids on the same page for when the ids are being added / read
func getId(ref any) []byte {
	arr := reflect.ValueOf(ref)
	arrLen := arr.Len()

	UID := make([]byte, arrLen)
	for i := 0; i < arrLen; i++ {
		UID[i] = arr.Index(i).Interface().(uint8)
	}

	return UID
}
