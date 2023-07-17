package folderscan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cbroglie/mustache"
	"github.com/metarex-media/mrx-tool/encode"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileExtract(t *testing.T) {

	empty := []byte(`{
		"cameras":
		{
			"CameraComponent":
			{
				"intrinsics": [ 5440.0002265929279, 0, 960, 0, 5440.0002265929279, 540, 0, 0, 1 ],
				"rotation": [ 0, 0, 0, 1 ],
				"translation": [ 0, 0, 0 ],
				"sensor_size": [ 1920, 1080 ]
			}
		},
		"extra" : "fill"
	}`)

	for i := 0; i < 144; i++ {
		f, _ := os.Create(fmt.Sprintf("./testdata/testbase2/0003Stream/%vd", i))
		f.Write(empty)
	}

	/*

		these tests need to check each bit of the chain works for a data input
		due to the nature the files should probably be generate before each test

		and one that does the errors

	*/
	targets := []string{"./testdata/testbase", "./testdata/testbase/flat"}

	for _, target := range targets {
		goodTest := folderScanner{folder: target}

		// run the test as if it was being run  by encode, checking each step of the process.
		Convey(fmt.Sprintf("Checking that each stage of the interface writer runs without error, using %v as the input folder", target), t, func() {
			Convey("checking the error of GetRoundTrip, then GetStreamInformation and EssencePipe", func() {
				Convey("No error is returned for each step", func() {
					_, err := goodTest.GetRoundTrip()
					So(err, ShouldBeNil)
					si, folderErr := goodTest.GetStreamInformation()
					So(folderErr, ShouldBeNil)
					fakePipes := make(chan *encode.ChannelPackets, si.ChannelCount)
					go pipeClear(fakePipes)
					StreamErr := goodTest.EssenceChannels(fakePipes)
					So(StreamErr, ShouldBeNil)
				})
			})
		})
	}
}

func pipeClear(pipes chan *encode.ChannelPackets) {
	for pipe := range pipes {
		_, openChannel := <-pipe.Packets
		for openChannel {
			_, openChannel = <-pipe.Packets
			/*
				just hoover out the channels and don't do anything
			*/
		}

	}

}

func TestEncodeErrors(t *testing.T) {
	// create some bodies to test the expected failures of encode without marrying it folder scan

	targets := []string{"./testdata/errors/not/a/real/location", "./testdata/errors/mixedEssence"}
	expectedErr := []string{
		"Error reading folder {{location}} : open {{location}}: The system cannot find the path specified.",
		"Mixed essence file types found in {{location}}, please ensure they are all the same type"}

	for i, target := range targets {
		target, _ = filepath.Abs(target)
		extractError := folderScanner{folder: target}
		errMessage, _ := mustache.Render(expectedErr[i], map[string]string{"location": target})

		// run the test as if it was being run  by encode, checking each step of the process.
		Convey(fmt.Sprintf("Checking that the inital parsing of the folder catches errors, using %v as the input folder", target), t, func() {
			Convey("checking the error of GetStreamInformation, with delibrate errors in them", func() {
				Convey(fmt.Sprintf("An error of %v is returned", errMessage), func() {
					_, err := extractError.GetRoundTrip()
					So(err, ShouldBeNil)
					_, folderErr := extractError.GetStreamInformation()
					So(folderErr.Error(), ShouldResemble, errMessage)

				})
			})
		})
	}

	pipeTarget := "./testdata/errors/deleted"
	target, _ := filepath.Abs("./testdata/errors/deleted/0000StreamTC0001d")
	pipeBreak := folderScanner{folder: pipeTarget}

	// write the deleted file
	f, _ := os.Create("./testdata/errors/deleted/0000StreamTC0001d")
	f.Write([]byte("{\"A test json\":\"designed to be deleted\"}"))
	f.Close()

	errMessage, _ := mustache.Render(
		"Error extracting data to encode from {{location}}:open {{location}}: The system cannot find the file specified.", map[string]string{"location": target})
	// run the test as if it was being run  by encode, checking each step of the process.

	Convey("Checking that the pipe errors are returned", t, func() {
		Convey("checking the error of GetRoundTrip, then GetStreamInformation and  then EssencePipe, expecting essence pipe to fail", func() {
			Convey("An error of "+errMessage+" is returned", func() {
				_, err := pipeBreak.GetRoundTrip()
				So(err, ShouldBeNil)
				si, folderErr := pipeBreak.GetStreamInformation()
				So(folderErr, ShouldBeNil)

				fakePipes := make(chan *encode.ChannelPackets, si.ChannelCount)
				//delete the file to invoke an error
				os.Remove("./testdata/errors/deleted/0000StreamTC0001d")

				go pipeClear(fakePipes)
				StreamErr := pipeBreak.EssenceChannels(fakePipes)
				So(StreamErr.Error(), ShouldResemble, errMessage)
			})
		})
	})

}
