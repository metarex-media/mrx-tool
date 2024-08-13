package mrxUnitTest

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/metarex-media/mrx-tool/klv"
	mxf2go "github.com/metarex-media/mxf-to-go"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type Node struct {
	Key, Length, Value Noder
	Properties         any
	// talk through the children role with Bruce
	// but keep as this
	Children []*Node
}

type Noder interface {
	Start() int
	End() int
}

type n struct {
	NStart, NEnd int
}

func (n n) Start() int { return n.NStart }
func (n n) End() int   { return n.NEnd }

type MXFControlNode struct {
	// each child is a partition from the mxf
	Children []*Node
	Primer   map[string]string //
}

type EssenceProperties struct {
	SID int
}

type GroupProperties struct {
	UUID mxf2go.TUUID
}

type PartitionProperties struct {
	PartitionCount int // the count of the partition along the MXF
	PartitionType  string
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

// inlcude the logger? if there's any errors flush them - discard ifo for unkown keys fro the moment
func MakeAST(stream io.Reader, dest io.Writer, buffer chan *klv.KLV, size int) (MXFControlNode, error) { // wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	// initiate the klv stream
	errs.Go(func() error {
		return klv.StartKLVStream(stream, buffer, size)
	})

	mxf := MXFControlNode{Children: make([]*Node, 0)}
	var currentNode *Node
	var currentPartition int
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
		// handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence
			if partitionName(klvItem.Key) == "060e2b34.020501  .0d010201.01    00" {
				if currentNode != nil {
					mxf.Children = append(mxf.Children, currentNode)
				}

				// add details from now
				currentNode = &Node{
					Key:      n{NStart: offset, NEnd: offset + len(klvItem.Key)},
					Length:   n{NStart: offset + len(klvItem.Key), NEnd: offset + len(klvItem.Key) + len(klvItem.Length)},
					Value:    n{NStart: offset + len(klvItem.Key) + len(klvItem.Length), NEnd: offset + klvItem.TotalLength()},
					Children: make([]*Node, 0),
				}

				// create a reference map for every node that is found
				refMap := make(map[*Node]refAndChild)
				offset += klvItem.TotalLength()
				// test the previous partitions essence as the final step
				// if len(contents.RipLayout) == 0 and the cache length !=0 emit an error that essence was found first

				partProps := PartitionProperties{PartitionCount: currentPartition}
				currentPartition++
				switch klvItem.Key[13] {
				case 17:
					partProps.PartitionType = "RIP"
				case 02:
					// header
					partProps.PartitionType = headerPartition
				case 03:
					// body
					if klvItem.Key[14] == 17 {
						partProps.PartitionType = genericStreamPartition
					} else {
						partProps.PartitionType = bodyPartition
					}
				case 04:
					// footer
					partProps.PartitionType = footerPartition
				default:
					// is nothing
					partProps.PartitionType = "invalid"

				}

				currentNode.Properties = partProps

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
						Key:    n{NStart: offset, NEnd: offset + len(flush.Key)},
						Length: n{NStart: offset + len(flush.Key), NEnd: offset + len(flush.Key) + len(flush.Length)},
						Value:  n{NStart: offset + len(flush.Key) + len(flush.Length), NEnd: offset + flush.TotalLength()},
					}

					refMap[flushNode] = refAndChild{}

					// @TODO include KLV fill packets
					dec, skip := decodeBuilder(flush.Key[5])
					flush.Key[5] = 0x7f
					if skip {

						//unpack the primer
						if fullName(flush.Key) == "060e2b34.027f0101.0d010201.01050100" {
							out := make(map[string]string)
							primerUnpack(flush.Value, out)
							mxf.Primer = out

						}
						// want to loop through them all?

					} else {

						decoders, ok := mxf2go.Groups["urn:smpte:ul:"+fullName(flush.Key)]

						if !ok {
							flush.Key[13] = 0x7f
							decoders, ok = mxf2go.Groups["urn:smpte:ul:"+fullName(flush.Key)]
						}

						// find the groups first
						pos := 0
						for pos < len(flush.Value) {
							key, klength := dec.keyFunc(flush.Value[pos : pos+dec.keyLen])
							length, lenlength := dec.lengthFunc(flush.Value[pos+dec.keyLen : pos+dec.keyLen+dec.lengthLen])
							if klength != 16 {
								key = mxf.Primer[key]
							}
							if key == "060e2b34.01010101.01011502.00000000" {
								out, _ := mxf2go.DecodeTUUID(flush.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
								flushNode.Properties = GroupProperties{UUID: out.(mxf2go.TUUID)}
								UUID := out.(mxf2go.TUUID)
								idMap[string(UUID[:])] = flushNode

							} else {

								if ok {
									// check the decoder for the field
									decodeF, ok := decoders.Group["urn:smpte:ul:"+key]

									if ok {

										b, _ := decodeF.Decode(flush.Value[pos+dec.keyLen+dec.lengthLen : pos+dec.keyLen+dec.lengthLen+length])
										strongRefs := StrongReference(b)
										if len(strongRefs) > 0 {
											mid := refMap[flushNode]
											mid.ref = append(mid.ref, strongRefs...)
											refMap[flushNode] = mid
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
						n.Children = append(n.Children, child)
					}
				}

				// then by assigning all the parents
				for n, refs := range refMap {

					if !refs.child {
						currentNode.Children = append(currentNode.Children, n)
					}
				}

			} else {

				essNode := &Node{
					Key:      n{NStart: offset, NEnd: offset + len(klvItem.Key)},
					Length:   n{NStart: offset + len(klvItem.Key), NEnd: offset + len(klvItem.Key) + len(klvItem.Length)},
					Value:    n{NStart: offset + len(klvItem.Key) + len(klvItem.Length), NEnd: offset + klvItem.TotalLength()},
					Children: make([]*Node, 0),
				}

				currentNode.Children = append(currentNode.Children, essNode)
				offset += klvItem.TotalLength()
				// throw a warning here saying expected partition got KEY : fullname

			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}
		mxf.Children = append(mxf.Children, currentNode)
		return nil
	})

	// post processing data if the klv hasn't returned an error
	// count of partitions
	errs.Wait()

	b, _ := yaml.Marshal(mxf)
	dest.Write(b)

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

func oneLengthKL(lengthbytes []byte) (int, int) {
	if len(lengthbytes) != 1 {
		return 0, 0
	}

	return int(lengthbytes[0]), 1
}

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

// map of UUID and parents
// if the uuid is found
// then assignt he child to the parents

// StrongRegerence checks if a type is strong reference,
// then recurisvely searches through the types to find the strong set version
func StrongReference(field any) [][]byte {

	switch v := field.(type) {
	case mxf2go.TStrongReference:
		return [][]byte{v}
	default:
		switch {
		case strings.Contains(reflect.TypeOf(field).Name(), "StrongReferenceSet") || strings.Contains(reflect.TypeOf(field).Name(), "StrongReferenceVector"):
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
				result := StrongReference(arrField)
				referenced[i] = result[0]
			}

			return referenced
		case strings.Contains(reflect.TypeOf(field).Name(), "StrongReference"):
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
