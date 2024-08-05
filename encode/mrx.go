package encode

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/metarex-media/mrx-tool/manifest"
	mxf2go "github.com/metarex-media/mxf-to-go"
)

// MrxWriter is the encoding engine for MRX files
type MrxWriter struct {
	writeInformation *writerInformation
	frameInformation *frameInformation
	// saver is the object
	// that handles the metadata streams
	// and lets the engine know what keys are in use etc/
	saver Writer
}

// writerInformation contains the time the file was made and the UMID
type writerInformation struct {
	mrxUMID       mxf2go.TPackageIDType
	buildTime     mxf2go.TTimeStamp
	buildTimeTime time.Time
}

// frameInformation holds the frame rate and total frame count of the file
type frameInformation struct {
	FrameRate     mxf2go.TRational
	TotalFrames   int
	ContainerKeys [][]byte
	// map[int]FrameRate
	StreamTimeLine manifest.Configuration
}

/*
type indexTable struct {
	layout  mxf2go.GIndexTableSegmentStruct
	offSets []int
}*/

// NewMRXWriterFR creates a Metarex file writer with the base frame rate.
// The frame rate string follows the format of %v/%v.
func NewMRXWriterFR(framRate string) (*MrxWriter, error) {

	var num, dom int32
	_, err := fmt.Sscanf(framRate, "%v/%v", &num, &dom)

	if err != nil {
		return nil, fmt.Errorf("error getting the frame rate from %v: %v", framRate, err)
	}

	return newWriter(num, dom)

}

// UpdateWriteMethod sets the write handler for the mrx file.
// Example writers include the EncodeSingleDataStream() function in
// the example folders.
func (mw *MrxWriter) UpdateWriteMethod(writeMethod Writer) {
	mw.saver = writeMethod
}

var order = binary.BigEndian

// newWriter generates a new mrx writer, frameNumerator and frameDenominator represent the frame rate.
// e.g. 24 fps is 24/1 where the numerator is 24 and the demoniator is 1. OR 29.97 fps is 30,000/1,001.
// The frame count is the total number of frames.
func newWriter(frameNumerator, frameDenominator int32) (*MrxWriter, error) {

	if frameNumerator == 0 {
		return nil, fmt.Errorf("The Numerator is  0, this is an invalid frame rate")
	}

	if frameDenominator == 0 {
		return nil, fmt.Errorf("The Denominator is  0, this is an invalid frame rate")
	}

	// rand.Seed(time.Now().Unix())

	// byte 11 is material type
	// byte 12 is the creation method 02 uuid for the top nibble
	// and no defined method for the bottom
	var smpteLabel = [12]byte{0x6, 0xa, 0x2b, 0x34, 0x01, 0x01, 0x01, 0x05, 0x01, 0x01, 0x0d, 0b00100000} // "060a2b340101010501010d00"}
	mxfUUID := uuid.New()

	Data4 := mxf2go.TUInt8Array8{}
	for i := range Data4 {
		Data4[i] = mxfUUID[8+i]
	}

	mat := mxf2go.TAUID{Data1: order.Uint32(mxfUUID[0:4]), Data2: order.Uint16(mxfUUID[4:6]), Data3: order.Uint16(mxfUUID[6:8]), Data4: Data4}

	wi := writerInformation{mrxUMID: mxf2go.TPackageIDType{SMPTELabel: smpteLabel, Length: 19, InstanceHigh: uint8(rand.Intn(0xff)),
		InstanceMid: uint8(rand.Intn(0xff)), InstanceLow: uint8(rand.Intn(0xff)), Material: mat}}

	fi := frameInformation{FrameRate: mxf2go.TRational{Numerator: frameNumerator, Denominator: frameDenominator}}

	return &MrxWriter{

			writeInformation: &wi,
			frameInformation: &fi,
		},
		nil
}

// NewMRXWriter generates a new MRX body for writing files.
func NewMRXWriter() *MrxWriter {
	var smpteLabel = [12]byte{0x6, 0xa, 0x2b, 0x34, 0x01, 0x01, 0x01, 0x05, 0x01, 0x01, 0x0d, 0b00100000} // "060a2b340101010501010d00"}
	mxfUUID := uuid.New()

	Data4 := mxf2go.TUInt8Array8{}
	for i := range Data4 {
		Data4[i] = mxfUUID[8+i]
	}

	mat := mxf2go.TAUID{Data1: order.Uint32(mxfUUID[0:4]), Data2: order.Uint16(mxfUUID[4:6]), Data3: order.Uint16(mxfUUID[6:8]), Data4: Data4}

	wi := writerInformation{mrxUMID: mxf2go.TPackageIDType{SMPTELabel: smpteLabel, Length: 19, InstanceHigh: uint8(rand.Intn(0xff)),
		InstanceMid: uint8(rand.Intn(0xff)), InstanceLow: uint8(rand.Intn(0xff)), Material: mat}}

	fi := frameInformation{}

	return &MrxWriter{

		writeInformation: &wi,
		frameInformation: &fi,
	}
}
