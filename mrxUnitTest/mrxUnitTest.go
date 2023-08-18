// mrxUnitTest runs tests on mrx files, to help locate errors within the mrx file
package mrxUnitTest

import (
	"context"
	"io"
	"os"

	"github.com/metarex-media/mrx-tool/klv"
	"golang.org/x/sync/errgroup"
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

func decode(stream io.Reader) error {

	klvChan := make(chan *klv.KLV, 1000)
	_, err := Decodeklv(stream, klvChan, 10)

	if err != nil {
		return err
	}

	return nil
}

// inlcude the logger? if there's any errors flush them - discard ifo for unkown keys fro the moment
func Decodeklv(stream io.Reader, buffer chan *klv.KLV, size int) (*MrxContents, error) { //wg *sync.WaitGroup, buffer chan packet, errChan chan error) {

	// use errs to handle errors while runnig concurrently
	errs, _ := errgroup.WithContext(context.Background())

	//initiate the klv stream
	errs.Go(func() error {
		return klv.BufferWrap(stream, buffer, size)
	})

	var contents layout
	f, _ := os.Create("test.log")
	contents.testLog = f

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

		//handle each klv packet
		for klvOpen {

			// check if it is a partition key
			// if not its presumed to be essence
			if partitionName(klvItem.Key) == "060e2b34.020501  .0d010201.01    00" {

				// test the previous partitions essence as the final step
				// if len(contents.RipLayout) == 0 and the cache length !=0 emit an error that essence was found first

				if len(contents.cache.keys) != 0 {
					contents.essenceTests(*contents.currentPartition)
				}

				if klvItem.Key[13] == 17 {
					// fmt.Println("RIP", klvItem.TotalLength())
					contents.TotalByteCount += klvItem.TotalLength()
					contents.ripHandle(klvItem)
					// handle the rip

					// then hoover the rest of the essence saying 25 bytes were found after the end of  file

				} else {
					// decode the partition - get the raw information out and handle the metadata
					// intermediate stage is binning of the metadata
					err := contents.partitionDecode(klvItem, buffer)

					if err != nil {
						//handle it
						return err
					}
				}
			} else {

				// throw a warning here saying expected partition got KEY : fullname

				contents.essenceCheck(klvItem)
				// some key handling

				/*
					seg testing function to be closed when the result is run
					a new test is generated for the keys of partition

					test the keys as they come in
					and stash some in the partition cache for later
				*/

				contents.TotalByteCount += klvItem.TotalLength()
				// decode the essence key - don't look in it what the data is
				/*

					get the key
					if making the frame include it in the sequence

					if the keys is recognised run additional checks - such as only one key in the clip wrapping etc element count


					else check it matches the position in the relative sequence

				*/

			}

			// get the next item for a loop
			klvItem, klvOpen = <-buffer
		}

		return nil
	})

	// collect any errors from the decode process
	err := errs.Wait()
	// fmt.Println(err, "potential error here")
	if err != nil {
		// log the fatal error
		// errors do not contribute to the testing process
		return nil, err
	}

	// post processing data if the klv hasn't returned an error
	// count of partitions

	return &MrxContents{}, nil
}

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
	FrameKeys       [][]byte // this is built along
	maxCount        int      // for clip wrapped this should be 1 or clipWrapped bool
	completeFrame   bool
	ParentPartition int // is this needed or will the layout be part of the proessing

}

type MrxContents struct {
	FrameWrapped []StreamContents
	ClipWrapped  []StreamContents

	header, footer any
}

// StreamContents contains the layout for a single dataStream
// it contains only one frames worth of essence KEys
// an error will have been returned if the keys do not follow the same pattern throughout.
type StreamContents struct {
	SID       int
	FrameKeys [][]byte //so i can discern the order
}

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
}
