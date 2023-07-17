package encode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/metarex-media/mrx-tool/essence"
	"gitlab.com/mm-eng/generatedmrx"
)

func (mw *MxfWriter) contentStorage(tag *uint16, tags map[string][]byte, stream mrxLayout) ([]byte, generatedmrx.TUUID) {
	var contentStorage bytes.Buffer

	// generate the timecodes

	timelineBytes, timeCodeID := mw.frameInformation.sourcePackageTimeline(tag, tags, generatedmrx.TPackageIDType{}, stream)

	// if mw.frameInformation.EssenceKeys contains the isxd essence descriptor flag
	// @TODO update this so isxd header is only called when the isxd key is used
	var mrxDescBytes []byte
	var mrxDescID generatedmrx.TUUID
	if stream.isxdflag {
		mrxDescBytes, mrxDescID = mw.frameInformation.isxdHeader(tag, tags, stream)
	} else {
		mrxDescID = generatedmrx.TUUID(uuid.New())
		desc := generatedmrx.GEssenceDescriptorStruct{InstanceID: mrxDescID}
		mrxDescBytes, _ = desc.Encode(tag, tags)

	}

	// file package generation here
	sourceInstanceId := generatedmrx.TUUID(uuid.New())
	sourcePackage := generatedmrx.GSourcePackageStruct{CreationTime: mw.writeInformation.buildTime, InstanceID: sourceInstanceId, PackageID: mw.writeInformation.mrxUMID,
		PackageTracks: timeCodeID, PackageLastModified: mw.writeInformation.buildTime, EssenceDescription: mrxDescID[:]}
	sourcePackageBytes, _ := sourcePackage.Encode(tag, tags)

	materialBytes, materialID := mw.frameInformation.materialPackage(tag, tags, mw.writeInformation.mrxUMID, stream)
	// then

	contentID := generatedmrx.TUUID(uuid.New())
	contentObj := generatedmrx.GContentStorageStruct{InstanceID: contentID, Packages: []generatedmrx.TPackageStrongReference{materialID[:], sourceInstanceId[:]}} // TPackageStrongReferenceSet figure out how packages are referenced
	contentObjByte, _ := contentObj.Encode(tag, tags)

	contentStorage.Write(contentObjByte)
	contentStorage.Write(materialBytes)
	contentStorage.Write(sourcePackageBytes)
	contentStorage.Write(timelineBytes)
	// contentStorage.Write(timelineBytesMP)
	contentStorage.Write(mrxDescBytes)

	return contentStorage.Bytes(), contentID
}

func (fi *frameInformation) outputTimeline(tag *uint16, tags map[string][]byte, UMID generatedmrx.TPackageIDType, stream mrxLayout) ([]byte, generatedmrx.TTrackStrongReferenceVector) {

	StrongReferences := make([]generatedmrx.TTrackStrongReference, 0)
	var timeLineBuffer bytes.Buffer

	// include the timecode as well
	var head bool

	//@TODO rest the tag each loop to prevent 100s of duplicates in the primer
	for _, str := range stream.dataStreams {

		if str.clocked && !head {

			//set up new tag here to stop repeats
			sourceClipID := generatedmrx.TUUID(uuid.New())
			sourceClip := generatedmrx.GSourceClipStruct{StartPosition: 0, InstanceID: sourceClipID, SourceTrackID: uint32(0), ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}, SourcePackageID: UMID}
			sourceClipBytes, _ := sourceClip.Encode(tag, tags)
			//060e2b34.04010101.01030202.03000000
			essenceSequenceID := generatedmrx.TUUID(uuid.New())
			essenceSequence := generatedmrx.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: generatedmrx.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqB, _ := essenceSequence.Encode(tag, tags)

			timeLineEssID := generatedmrx.TUUID(uuid.New())
			timeLineEss := generatedmrx.GTimelineTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 0}
			timeEssCB, _ := timeLineEss.Encode(tag, tags)

			timeLineBuffer.Write(timeEssCB)
			timeLineBuffer.Write(essSeqB)
			timeLineBuffer.Write(sourceClipBytes)

			// set up the time code
			timeCodeID := generatedmrx.TUUID(uuid.New())
			timeCode := generatedmrx.GTimecodeStruct{StartTimecode: 0, InstanceID: timeCodeID, FramesPerSecond: uint16(str.frameRate.Numerator),
				ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}}
			timeCodeBytes, _ := timeCode.Encode(tag, tags)
			//060e2b34.04010101.01030202.03000000
			essenceSequenceTCID := generatedmrx.TUUID(uuid.New())
			essenceSequenceTC := generatedmrx.GSequenceStruct{InstanceID: essenceSequenceTCID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: generatedmrx.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqTCB, _ := essenceSequenceTC.Encode(tag, tags)

			timeLineEssTCID := generatedmrx.TUUID(uuid.New())
			timeLineEssTC := generatedmrx.GTimelineTrackStruct{InstanceID: timeLineEssTCID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 0}
			timeEssTCB, _ := timeLineEssTC.Encode(tag, tags)

			timeLineBuffer.Write(timeEssTCB)
			timeLineBuffer.Write(essSeqTCB)
			timeLineBuffer.Write(timeCodeBytes)

			StrongReferences = append(StrongReferences, timeLineEssID[:])

			head = true
		}
	}

	return timeLineBuffer.Bytes(), StrongReferences
}

func (fi *frameInformation) sourcePackageTimeline(tag *uint16, tags map[string][]byte, UMID generatedmrx.TPackageIDType, stream mrxLayout) ([]byte, generatedmrx.TTrackStrongReferenceVector) {
	StrongReferences := make([]generatedmrx.TTrackStrongReference, 0)
	var timeLineBuffer bytes.Buffer

	// generate a  timecode track as well
	// as well as a source key one

	// keys := orderKeys(fi.StreamTimeLine.StreamProperties)
	// each  stream needs a timeline? do this then refine
	// rest the tag each loop to prevent 100s of duplicates in the primer

	var head bool
	var staticTracks []generatedmrx.TComponentStrongReference
	// Sid is 2 because that is the first static track
	// if no tracks are present then this won't be used
	sid := uint32(2)

	for _, str := range stream.dataStreams {

		if str.clocked && !head {
			//
			fi.FrameRate = str.frameRate
			// set up new tag here to stop repeats
			sourceClipID := generatedmrx.TUUID(uuid.New())
			sourceClip := generatedmrx.GSourceClipStruct{StartPosition: 0, InstanceID: sourceClipID, SourceTrackID: uint32(0), ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00}, SourcePackageID: UMID}
			sourceClipBytes, _ := sourceClip.Encode(tag, tags)
			//060e2b34.04010101.01030202.03000000
			essenceSequenceID := generatedmrx.TUUID(uuid.New())
			essenceSequence := generatedmrx.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
				ComponentObjects: generatedmrx.TComponentStrongReferenceVector{sourceClipID[:]}}
			essSeqB, _ := essenceSequence.Encode(tag, tags)

			timeLineEssID := generatedmrx.TUUID(uuid.New())
			timeLineEss := generatedmrx.GTimelineTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
				EditRate: str.frameRate, Origin: 0,
				TrackSegment: essenceSequenceID[:], EssenceTrackNumber: order.Uint32(str.key[12:])} // @TODO update the track number with something sensible
			timeEssCB, _ := timeLineEss.Encode(tag, tags)

			timeLineBuffer.Write(timeEssCB)
			timeLineBuffer.Write(essSeqB)
			timeLineBuffer.Write(sourceClipBytes)

			StrongReferences = append(StrongReferences, timeLineEssID[:])

			head = true
		} else if !str.clocked {

			metaDataID := generatedmrx.TUUID(uuid.New())
			metaDataSet := generatedmrx.GGenericStreamTextBasedSetStruct{InstanceID: metaDataID, GenericStreamID: sid,
				TextMIMEMediaType: []rune("application/octet-stream"), RFC5646TextLanguageCode: []rune("en"),
				// 060E2B34.0401010C.0D010401.04010100
				TextBasedMetadataPayloadSchemeID: generatedmrx.TAUID{Data1: order.Uint32([]byte{06, 0xe, 0x2b, 0x34}),
					Data2: order.Uint16([]byte{04, 01}), Data3: order.Uint16([]byte{01, 0x0c}),
					Data4: generatedmrx.TUInt8Array8{0xd, 01, 04, 01, 04, 01, 01, 00}}}
			metaDataSetBytes, _ := metaDataSet.Encode(tag, tags)

			frameID := generatedmrx.TUUID(uuid.New())
			frame := generatedmrx.GTextBasedFrameworkStruct{InstanceID: frameID, TextBasedObject: metaDataID[:]}
			frameBytes, _ := frame.Encode(tag, tags)

			// en as the default
			descID := generatedmrx.TUUID(uuid.New())
			// inactive user bits	060e2b34.04010101.01030201.01000000
			descSequence := generatedmrx.GDescriptiveMarkerStruct{InstanceID: descID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 01, 01, 00, 00, 00},
				DescriptiveFrameworkObject: frameID[:]}
			//	essenceSequence := generatedmrx.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 02, 03, 00, 00, 00},
			//		ComponentObjects: generatedmrx.TComponentStrongReferenceVector{sourceClipID[:]}}
			descSeqB, _ := descSequence.Encode(tag, tags)

			timeLineBuffer.Write(descSeqB)
			timeLineBuffer.Write(frameBytes)
			timeLineBuffer.Write(metaDataSetBytes)

			staticTracks = append(staticTracks, descID[:])

			sid++
		}
	}

	// if there are static tracks add them to the single static track sequence object
	if len(staticTracks) != 0 {

		essenceSequenceID := generatedmrx.TUUID(uuid.New())
		essenceSequence := generatedmrx.GSequenceStruct{InstanceID: essenceSequenceID, ComponentDataDefinition: []byte{0x06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 01, 03, 02, 01, 01, 00, 00, 00},
			ComponentObjects: staticTracks}
		essSeqB, _ := essenceSequence.Encode(tag, tags)

		timeLineEssID := generatedmrx.TUUID(uuid.New())
		timeLineEss := generatedmrx.GStaticTrackStruct{InstanceID: timeLineEssID, TrackID: uint32(0),
			TrackSegment: essenceSequenceID[:], EssenceTrackNumber: 1} // @TODO update the track number with something sensible
		timeEssCB, _ := timeLineEss.Encode(tag, tags)

		timeLineBuffer.Write(timeEssCB)
		timeLineBuffer.Write(essSeqB)

		StrongReferences = append(StrongReferences, timeLineEssID[:])

	}

	return timeLineBuffer.Bytes(), StrongReferences
}

func (fi *frameInformation) materialPackage(tag *uint16, tags map[string][]byte, UMID generatedmrx.TPackageIDType, stream mrxLayout) ([]byte, generatedmrx.TUUID) {

	// get the time code for the putput of the file
	timelineBytes, timeCodeID := fi.outputTimeline(tag, tags, generatedmrx.TPackageIDType{}, stream)

	materialID := generatedmrx.TUUID(uuid.New())
	gTime := time.Now()
	Date := generatedmrx.TDateStruct{Year: int16(gTime.Year()), Month: uint8(gTime.Month()), Day: uint8(gTime.Day())}
	Time := generatedmrx.TTimeStruct{Hour: uint8(gTime.Hour()), Minute: uint8(gTime.Minute()), Second: uint8(gTime.Second())}

	materialPack := generatedmrx.GMaterialPackageStruct{InstanceID: materialID, PackageID: UMID,
		PackageLastModified: generatedmrx.TTimeStamp{Date: Date, Time: Time}, CreationTime: generatedmrx.TTimeStamp{Date: Date, Time: Time},
		PackageTracks: timeCodeID}

	materialPackBytes, _ := materialPack.Encode(tag, tags)

	return append(materialPackBytes, timelineBytes...), materialID
}

func (fi *frameInformation) isxdHeader(tag *uint16, tags map[string][]byte, streamLayout mrxLayout) ([]byte, generatedmrx.TUUID) {
	var isxdBuffer bytes.Buffer

	isxdID := generatedmrx.TUUID(uuid.New())
	// @TODO make the namespace to be changeable
	// and add a more sensible default
	b := nameSpaces(streamLayout)

	ISXD := generatedmrx.GISXDStruct{NamespaceURIUTF8: []rune(string(b)), InstanceID: isxdID, SampleRate: fi.FrameRate,
		ContainerFormat: []byte{0x06, 0x0e, 0x2b, 0x34, 0x04, 0x01, 0x01, 0x05, 0x0e, 0x09, 0x06, 0x07, 0x01, 0x01, 0x01, 0x03}, DataEssenceCoding: generatedmrx.TAUID{Data1: 0x060E2B34, Data2: 0x0401, Data3: 0x0105, Data4: [8]byte{0x0e, 0x09, 06, 06, 00, 00, 00, 00}}}
	isb, _ := ISXD.Encode(tag, tags)

	isxdBuffer.Write(isb)

	TextBasedDmFramework := generatedmrx.GTextBasedFrameworkStruct{InstanceID: generatedmrx.TUUID(uuid.New())}
	TextBasedDmFrameworkb, _ := TextBasedDmFramework.Encode(tag, tags)

	isxdBuffer.Write(TextBasedDmFrameworkb)

	return isxdBuffer.Bytes(), isxdID
}

func nameSpaces(bases mrxLayout) []byte {

	nameSpaces := make(map[string]string)
	sidCount := 2

	for _, bp := range bases.dataStreams {

		channelID := essence.FullName(bp.key)

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
var productID = generatedmrx.TAUID{Data1: 0xa28c37aa, Data2: 0x3b9a, Data3: 0x471e,
	Data4: generatedmrx.TUInt8Array8{0xa7, 0x4b, 0x84, 0x08, 0x03, 0xb0, 0xff, 0x1e}}

func newAUID() generatedmrx.TAUID {

	// auid is a swapping of the top and bottom bytes
	// pg 18 of 377
	// swasps dont's happen for
	// LinkedGenerationID
	// GenerationID
	// ApplicationProductID https://registry.smpte-ra.org/view/draft/docs/Register%20(Types)/Individual%20Types%20entries%20(EXCEPTIONS%20etc)/

	AUID := uuid.New()

	var array8 [8]uint8

	for j, arr := range AUID[8:] {
		array8[j] = arr
	}

	tauidKey := generatedmrx.TAUID{
		Data1: order.Uint32(AUID[0:4]),
		Data2: order.Uint16(AUID[4:6]),
		Data3: order.Uint16(AUID[6:8]),
		Data4: generatedmrx.TUInt8Array8(array8),
	}

	return tauidKey
}

func identification(tag *uint16, tags map[string][]byte) ([]byte, generatedmrx.TUUID) {
	idid := generatedmrx.TUUID(uuid.New())
	identifier := generatedmrx.GIdentificationStruct{InstanceID: idid, ApplicationSupplierName: []rune("metarex.media"), ApplicationName: []rune("MRX Tool"),
		ApplicationVersionString: []rune("0.0.1"), ApplicationProductID: productID, GenerationID: newAUID()}

	idb, _ := identifier.Encode(tag, tags)
	return idb, idid
}

func (wi *writerInformation) mrxEssenceDescriptor(tag *uint16, tags map[string][]byte) ([]byte, generatedmrx.TUUID) {
	mrxID := generatedmrx.TUUID(uuid.New())
	identifier := generatedmrx.GMRXessencedescriptorStruct{ISO8601Time: []rune(wi.buildTimeTime.Format("2006-01-02T15:04:05Z")),
		MetarexID: []rune("MRX.123.456.789.def"), RegURI: []rune("https://metarex.media/reg/"),
		InstanceID: mrxID}

	idb, _ := identifier.Encode(tag, tags)

	return idb, mrxID
}

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
	return [16]byte{06, 0x0e, 0x2b, 0x34, 04, 01, 01, 01, 0x0d, 01, 02, 01, 01, 01, 0b0101, 00} //01,02,03 then 01,02,03
	// 0b1
	// then 0b00 as we don't have external essence
	// then 0b100 as it is not a stream file
	// then 0b0000 as everything is uni track
	//last byte is set to 0
}
