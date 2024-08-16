// mrxUnitTest runs tests on mrx files, to help locate errors within the mrx file.
// @TODO finish writing and intergrating the code into the CLI.
// This package is very experimental and should not be used,
// as the API design hasn't even been finished.
package mrxUnitTest

import (
	"context"
	"io"
)

// set up a utility bit for the decodes?

var Prefix = "Byte Offset %v"

// Full error message warning Byte Offset 45 malformed partition header

/*

	klvChan := make(chan *klv.KLV, 1000)
	err := DecodeKLVToFile(stream, klvChan, parentFolder, flat, leadingZeros)

	if err != nil {
		return err
	}

	return nil


either the decoder handles all the logging of errors or the user flushes it.
flush all the errors are return nil if no errors were flushed


func Decodeklv(stream io.Reader, buffer chan *klv.KLV, size int, error logger) (*MrxContents, error) { //wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	//initiate the klv stream
	errs.Go(func() error {
		return klv.BufferWrap(stream, buffer, size)

	})



	go through similar to this decode method

	go through having a current partition object
	history object this logs position to compare against a rip pack (if there is any)





*/

/*

within the loop have a if (catastrophic error stop this all)

each bit should be able to read without failing and then process the contents to ensure they are correct.

Each step should have some error checking:
- essence keys - are these metarex keys? or any valid key check the elemenet count and number are maintained
- either partition
- header metadata is a different kettle of fish




*/

type layout struct {
	currentPartPos   int
	currentPartition *mxfPartition
	// log of partitions []array -> for comparing with the rip - also count footer
	// and headers etc and generic stream partition
	// current key layout map[essenceKeys]incase a streamID is replaced

	// MRX Contents
	TotalByteCount int

	// completed tests body here There needs to be this
	Rip []RIP

	Cache *context.Context // any

	cache essenceCache
	/*
		things to cache:
			partition positions (the rip)
			essence per partition (which is then removed per partition)
			the primer pack

	*/
	// error save destination
	// @TODO upgrade so that writers are dispersed to preserve the order
	// add some methods new writer branch or the likes
	testLog io.Writer
}

type EssenceKeys struct {
	FrameKeys [][]byte // this is built along
	//	maxCount        int      // for clip wrapped this should be 1 or clipWrapped bool
	//	completeFrame   bool
	ParentPartition int // is this needed or will the layout be part of the proessing

}

type MrxContents struct {
	FrameWrapped []StreamContents
	ClipWrapped  []StreamContents

	// header, footer any
}

// StreamContents contains the layout for a single dataStream
// it contains only one frames worth of essence KEys
// an error will have been returned if the keys do not follow the same pattern throughout.
type StreamContents struct {
	SID       int
	FrameKeys [][]byte // so i can discern the order
}

/*
func essHandle(*klv.KLV) {
	/*
	   get the essence, see if can be identified.

	   if its metarex than do some extra checks
	   frame wrapped:
	   - check the element count and number line up
	   - check the frame positions remain constant, no shifting essence

	   clip wrapped:
	   - check there is only one key
	   - check the partition key is generic partition
*/
//}
