package encode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/metarex-media/mrx-tool/decode"
	mxf2go "github.com/metarex-media/mxf-to-go"
)

func (mw *MrxWriter) contentStorage(primer *mxf2go.Primer, stream mrxLayout) ([]byte, mxf2go.TUUID) {
	var contentStorage bytes.Buffer

	// generate the timecodes

	timelineBytes, timeCodeID := mw.frameInformation.sourcePackageTimeline(primer, mxf2go.TPackageIDType{}, stream)

	// if mw.frameInformation.EssenceKeys contains the isxd essence descriptor flag
	// @TODO update this so isxd header is only called when the isxd key is used
	var mrxDescBytes []byte
	var mrxDescID mxf2go.TUUID
	if stream.isxdflag {
		mrxDescBytes, mrxDescID = mw.frameInformation.isxdHeader(primer, stream)
	} else {
		mrxDescID = mxf2go.TUUID(uuid.New())
		desc := mxf2go.GEssenceDescriptorStruct{InstanceID: mrxDescID}
		mrxDescBytes, _ = desc.Encode(primer)

	}

	// file package generation here
	sourceInstanceId := mxf2go.TUUID(uuid.New())
	sourcePackage := mxf2go.GSourcePackageStruct{CreationTime: mw.writeInformation.buildTime, InstanceID: sourceInstanceId, PackageID: mw.writeInformation.mrxUMID,
		PackageTracks: timeCodeID, PackageLastModified: mw.writeInformation.buildTime, EssenceDescription: mrxDescID[:]}
	sourcePackageBytes, _ := sourcePackage.Encode(primer)

	materialBytes, materialID := mw.frameInformation.materialPackage(primer, mw.writeInformation.mrxUMID, stream)
	// then

	contentID := mxf2go.TUUID(uuid.New())
	contentObj := mxf2go.GContentStorageStruct{InstanceID: contentID, Packages: []mxf2go.TPackageStrongReference{materialID[:], sourceInstanceId[:]}} // TPackageStrongReferenceSet figure out how packages are referenced
	contentObjByte, _ := contentObj.Encode(primer)

	contentStorage.Write(contentObjByte)
	contentStorage.Write(materialBytes)
	contentStorage.Write(sourcePackageBytes)
	contentStorage.Write(timelineBytes)
	// contentStorage.Write(timelineBytesMP)
	contentStorage.Write(mrxDescBytes)

	return contentStorage.Bytes(), contentID
}

func (fi *frameInformation) outputTimeline(primer *mxf2go.Primer, umid mxf2go.TPackageIDType, stream mrxLayout) ([]byte, mxf2go.TTrackStrongReferenceVector) {

	StrongReferences := make([]mxf2go.TTrackStrongReference, 0)
	var timeLineBuffer bytes.Buffer

	// include the timecode as well
	var head bool

	// @TODO rest the tag each loop to prevent 100s of duplicates in the primer
	for _, str := range stream.dataStreams {

		if str.clocked && !head {

			// set up new tag here to stop repeats
			sourceClipID := mxf2go.TUUID(uuid.New())
			sourceClip := mxf2go.GSourceClipStruct{StartPosition: 0, InstanceID: sourceClipID, SourceTrackID: uint32(0), ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}, SourcePackageID: umid}
			sourceClipBytes, _ := sourceClip.Encode(primer)
			// 060e2b34.04010101.01030202.03000000
			essenceSequenceID := mxf2go.TUUID(uuid.New())
			essenceSequence := mxf2go.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: mxf2go.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqB, _ := essenceSequence.Encode(primer)

			timeLineEssID := mxf2go.TUUID(uuid.New())
			timeLineEss := mxf2go.GTimelineTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 0}
			timeEssCB, _ := timeLineEss.Encode(primer)

			timeLineBuffer.Write(timeEssCB)
			timeLineBuffer.Write(essSeqB)
			timeLineBuffer.Write(sourceClipBytes)

			// set up the time code
			timeCodeID := mxf2go.TUUID(uuid.New())
			timeCode := mxf2go.GTimecodeStruct{StartTimecode: 0, InstanceID: timeCodeID, FramesPerSecond: uint16(str.frameRate.Numerator),
				ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}}
			timeCodeBytes, _ := timeCode.Encode(primer)
			// 060e2b34.04010101.01030202.03000000
			essenceSequenceTCID := mxf2go.TUUID(uuid.New())
			essenceSequenceTC := mxf2go.GSequenceStruct{InstanceID: essenceSequenceTCID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: mxf2go.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqTCB, _ := essenceSequenceTC.Encode(primer)

			timeLineEssTCID := mxf2go.TUUID(uuid.New())
			timeLineEssTC := mxf2go.GTimelineTrackStruct{InstanceID: timeLineEssTCID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 0}
			timeEssTCB, _ := timeLineEssTC.Encode(primer)

			timeLineBuffer.Write(timeEssTCB)
			timeLineBuffer.Write(essSeqTCB)
			timeLineBuffer.Write(timeCodeBytes)

			StrongReferences = append(StrongReferences, timeLineEssID[:])

			head = true
		}
	}

	return timeLineBuffer.Bytes(), StrongReferences
}

func (fi *frameInformation) sourcePackageTimeline(primer *mxf2go.Primer, umid mxf2go.TPackageIDType, stream mrxLayout) ([]byte, mxf2go.TTrackStrongReferenceVector) {
	StrongReferences := make([]mxf2go.TTrackStrongReference, 0)
	var timeLineBuffer bytes.Buffer

	// generate a  timecode track as well
	// as well as a source key one

	// keys := orderKeys(fi.StreamTimeLine.StreamProperties)
	// each  stream needs a timeline? do this then refine
	// rest the tag each loop to prevent 100s of duplicates in the primer

	var head bool
	var staticTracks []mxf2go.TComponentStrongReference
	// Sid is 2 because that is the first static track
	// if no tracks are present then this won't be used
	sid := uint32(2)

	for _, str := range stream.dataStreams {

		if str.clocked && !head {
			//
			fi.FrameRate = str.frameRate
			// set up new tag here to stop repeats
			sourceClipID := mxf2go.TUUID(uuid.New())
			sourceClip := mxf2go.GSourceClipStruct{StartPosition: 0, InstanceID: sourceClipID, SourceTrackID: uint32(0), ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}, SourcePackageID: umid}
			sourceClipBytes, _ := sourceClip.Encode(primer)
			// 060e2b34.04010101.01030202.03000000
			essenceSequenceID := mxf2go.TUUID(uuid.New())
			essenceSequence := mxf2go.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: mxf2go.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqB, _ := essenceSequence.Encode(primer)

			timeLineEssID := mxf2go.TUUID(uuid.New())
			timeLineEss := mxf2go.GTimelineTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: order.Uint32(str.key[12:])} // @TODO update the track number with something sensible
			timeEssCB, _ := timeLineEss.Encode(primer)

			timeLineBuffer.Write(timeEssCB)
			timeLineBuffer.Write(essSeqB)
			timeLineBuffer.Write(sourceClipBytes)

			StrongReferences = append(StrongReferences, timeLineEssID[:])

			head = true
		} else if !str.clocked {

			metaDataID := mxf2go.TUUID(uuid.New())
			metaDataSet := mxf2go.GGenericStreamTextBasedSetStruct{InstanceID: metaDataID, GenericStreamID: sid,
				TextMIMEMediaType: []rune("application/octet-stream"), RFC5646TextLanguageCode: []rune("en"),
				// 060E2B34.0401010C.0D010401.04010100
				TextBasedMetadataPayloadSchemeID: mxf2go.TAUID{Data1: order.Uint32([]byte{06, 0xe, 0x2b, 0x34}),
					Data2: order.Uint16([]byte{04, 01}), Data3: order.Uint16([]byte{01, 0x0c}),
					Data4: mxf2go.TUInt8Array8{0xd, 01, 04, 01, 04, 01, 01, 00}}}
			metaDataSetBytes, _ := metaDataSet.Encode(primer)

			frameID := mxf2go.TUUID(uuid.New())

			frame := mxf2go.GTextBasedFrameworkStruct{InstanceID: frameID, TextBasedObject: metaDataID[:]}
			frameBytes, _ := frame.Encode(primer)

			// en as the default
			descID := mxf2go.TUUID(uuid.New())
			// inactive user bits	060e2b34.04010101.01030201.01000000
			descSequence := mxf2go.GDescriptiveMarkerStruct{InstanceID: descID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 01, 01, 00, 00, 00},
				DescriptiveFrameworkObject: frameID[:]}
			//	essenceSequence := mxf2go.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
			//		ComponentObjects: mxf2go.TComponentStrongReferenceVector{sourceClipID[:]}}
			descSeqB, _ := descSequence.Encode(primer)

			timeLineBuffer.Write(descSeqB)
			timeLineBuffer.Write(frameBytes)
			timeLineBuffer.Write(metaDataSetBytes)

			staticTracks = append(staticTracks, descID[:])

			sid++
		}
	}

	// if there are static tracks add them to the single static track sequence object
	if len(staticTracks) != 0 {

		essenceSequenceID := mxf2go.TUUID(uuid.New())
		essenceSequence := mxf2go.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 01, 01, 00, 00, 00},
			ComponentObjects: staticTracks}
		essSeqB, _ := essenceSequence.Encode(primer)

		timeLineEssID := mxf2go.TUUID(uuid.New())
		timeLineEss := mxf2go.GStaticTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
			TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 1} // @TODO update the track number with something sensible
		timeEssCB, _ := timeLineEss.Encode(primer)

		timeLineBuffer.Write(timeEssCB)
		timeLineBuffer.Write(essSeqB)

		StrongReferences = append(StrongReferences, timeLineEssID[:])

	}

	return timeLineBuffer.Bytes(), StrongReferences
}

func (fi *frameInformation) materialPackage(primer *mxf2go.Primer, umid mxf2go.TPackageIDType, stream mrxLayout) ([]byte, mxf2go.TUUID) {

	// get the time code for the putput of the file
	timelineBytes, timeCodeID := fi.outputTimeline(primer, mxf2go.TPackageIDType{}, stream)

	materialID := mxf2go.TUUID(uuid.New())
	gTime := time.Now()
	Date := mxf2go.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := mxf2go.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	materialPack := mxf2go.GMaterialPackageStruct{InstanceID: materialID, PackageID: umid,
		PackageLastModified: mxf2go.TTimeStamp{Date: Date, Time: Time}, CreationTime: mxf2go.TTimeStamp{Date: Date, Time: Time},
		PackageTracks: timeCodeID}

	materialPackBytes, _ := materialPack.Encode(primer)

	return append(materialPackBytes, timelineBytes...), materialID
}

func (fi *frameInformation) isxdHeader(primer *mxf2go.Primer, streamLayout mrxLayout) ([]byte, mxf2go.TUUID) {
	var isxdBuffer bytes.Buffer

	isxdID := mxf2go.TUUID(uuid.New())
	// @TODO make the namespace to be changeable
	// and add a more sensible default
	b := nameSpaces(streamLayout)

	ISXD := mxf2go.GISXDStruct{NamespaceURIUTF8: []rune(string(b)), InstanceID: isxdID, SampleRate: fi.FrameRate,
		ContainerFormat: []byte{0x06, 0x0e, 0x2b, 0x34, 0x04, 0x01, 0x01, 0x05, 0x0e, 0x09, 0x06, 0x07, 0x01, 0x01, 0x01, 0x03}, DataEssenceCoding: mxf2go.TAUID{Data1: 0x060E2B34, Data2: 0x0401, Data3: 0x0105, Data4: [8]byte{0x0e, 0x09, 06, 06, 00, 00, 00, 00}}}
	isb, _ := ISXD.Encode(primer)

	isxdBuffer.Write(isb)

	TextBasedDmFramework := mxf2go.GTextBasedFrameworkStruct{InstanceID: mxf2go.TUUID(uuid.New())}
	TextBasedDmFrameworkb, _ := TextBasedDmFramework.Encode(primer)
	isxdBuffer.Write(TextBasedDmFrameworkb)

	return isxdBuffer.Bytes(), isxdID
}

func nameSpaces(bases mrxLayout) []byte {

	nameSpaces := make(map[string]string)
	sidCount := 2

	for _, bp := range bases.dataStreams {

		channelID := decode.FullName(bp.key)

		if bp.clocked {

			channelID += fmt.Sprintf(".%04d", 1)
		} else {
			channelID += fmt.Sprintf(".%04d", sidCount)
			sidCount++
		}

		nameSpaces[channelID] = bp.nameSpace

	}

	b, _ := json.Marshal(nameSpaces)
	return b
}

// a28c37aa-3b9a-471e-a74b-840803b0ff1e
var productID = mxf2go.TAUID{Data1: 0xa28c37aa, Data2: 0x3b9a, Data3: 0x471e,
	Data4: mxf2go.TUInt8Array8{0xa7, 0x4b, 0x84, 0x08, 0x03, 0xb0, 0xff, 0x1e}}

// NewAUID generates the AUID
func newAUID() mxf2go.TAUID {

	// auid is a swapping of the top and bottom bytes
	// pg 18 of 377
	// swasps dont's happen for
	// LinkedGenerationID
	// GenerationID
	// ApplicationProductID https://registry.smpte-ra.org/view/draft/docs/Register%20(Types)/Individual%20Types%20entries%20(EXCEPTIONS%20etc)/

	AUID := uuid.New()

	var array8 [8]uint8

	copy(array8[:], AUID[8:])

	tauidKey := mxf2go.TAUID{
		Data1: order.Uint32(AUID[0:4]),
		Data2: order.Uint16(AUID[4:6]),
		Data3: order.Uint16(AUID[6:8]),
		Data4: mxf2go.TUInt8Array8(array8),
	}

	return tauidKey
}

func identification(primer *mxf2go.Primer) ([]byte, mxf2go.TUUID) {
	idid := mxf2go.TUUID(uuid.New())
	identifier := mxf2go.GIdentificationStruct{InstanceID: idid, ApplicationSupplierName: []rune("metarex.media"), ApplicationName: []rune("MRX Tool"),
		ApplicationVersionString: []rune("0.0.1"), ApplicationProductID: productID, GenerationID: newAUID()}

	idb, _ := identifier.Encode(primer)
	return idb, idid
}

/*
func (wi *writerInformation) mrxEssenceDescriptor(tag *uint16, tags map[string][]byte) ([]byte, mxf2go.TUUID) {
	mrxID := mxf2go.TUUID(uuid.New())
	identifier := mxf2go.GMRXessencedescriptorStruct{ISO8601Time: []rune(wi.buildTimeTime.Format("2006-01-02T15:04:05Z")),
		MetarexID: []rune("MRX.123.456.789.def"), RegURI: []rune("https://metarex.media/reg/"),
		InstanceID: mrxID}

	idb, _ := identifier.Encode(primer)

	return idb, mrxID
}*/

// generateOperationalPattern gives the pattern of the mrx file
// this wil be updated to be adjustable baased on the inout as file complexity grows.
func generateOperationalPattern() [16]byte {

	/*
		Pattern
		1	single Item the source clip matches the length of this

		2	playlist everything is the same length but there's lots of items
		3   Edit item gives a lot of source but we aren't worried about that yet


		Complexity

		a single package, material package can only a ceese one fiule package
		b ganged pacakges. The material package can access one or more top level file packages
		c alternate packages. There are tow or more alternative material packages. Different material packages for different time zones


	*/
	// if one material its 1a
	// so far we only have one file package
	// Else its 1c as the data is carried through and accross

	// These packages water down to the simplest ones
	//
	return [16]byte{06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 0x0d, 01, 02, 01, 01, 01, 0b0101, 00} // 01,02,03 then 01,02,03
	// 0b1
	// then 0b00 as we don't have external essence
	// then 0b100 as it is not a stream file
	// then 0b0000 as everything is uni track
	// last byte is set to 0
}
