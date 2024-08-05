# MRX Tool

MRX Tool is a utility tool for interacting with [Metarex](https://metarex.media/) MRX files.
It is used for encoding and decoding MRX files,
with a focus on improving the transparency of what makes an mrx file.
If you need help understanding mrx files, or want to make your own mrx files
and don't know where to start, then this is the tool for you.

## Contents

- [Overview](#overview)
- [Installation](#installation)
- [Functions](#mrx-tool-functions)
  - [Decoding the mrx structure](#the-decode-flag)
  - [Extracting the metadata from mrx](#the-decodesave-flag)
  - [Encoding metadata to mrx](#the-encode-flag)
  - [The split flag](#the-split-flag)
- [Yaml Layout](#yaml-layout)
- [Notes for developers](#notes-for-developers)

## Overview

This tool is designed for the following use cases.

- validating an mrx file against the [Metarex specification](https://metarex.media/docs/specifications/),
this is still a work in progress.
- [decoding](#the-decode-flag) mrx files into the constituent metadata files
- [decoding](#the-decodesave-flag) the structural layout of an mrx file in json or yaml form
- [encoding](#the-encode-flag) metadata file(s) into a single mrx file

Using this tool allows you to understand the contents and requirements of the mrx files
and it can let you generate mrx test vectors.

Please check out the [demo repo](https://github.com/metarex-media/mrx-demo),
which gives an introduction and walkthrough of the features and how to use them.

## Installation

Please get the latest version for your operating system from the downloads page.
Or alternatively, download the repo and compile using the following command.
Make sure you have the latest version of Go from the [official golang
source][g1] installed.

```cmd
go build
```

This tool only runs on the command line.

## MRX Tool functions

The following commands are available.

- [decoding](#the-decode-flag) the structural layout of an mrx file in yaml form
- [decoding](#the-decodesave-flag) mrx files into the metadata sub components
- [encoding](#the-encode-flag) metadata file(s) into a single mrx file

### The decode flag

The decode flag breaks down the selected mrx file into a [yaml file](#yaml-layout), detailing the
labels of its contents and the overall file structure.

The `--input` and `--output` flags must be used for selecting the mrx file 
and saving the output file, it also contains an optional
[--split flag](#the-split-flag), if you would like to reduce the size of the yaml file.
The split flag separates the metadata segments into groups to be read and groups to be skipped.

Run the following commands as an example of the split function, this generates
one structure file with the split `result/rexy_sunbathe_mrx_Split.yaml` and
one without `result/rexy_sunbathe_mrx.yaml`.

```console
./mrx-tool.exe decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx.yaml
./mrx-tool.exe decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx_Split.yaml --split 3,5,6,7,3
```

Check the difference between the two yaml files that you
have generated, notice the size difference (24kb to 10kb) and the whole structure
has only been decoded for `result/rexy_sunbathe_mrx.yaml`. Count the number of
skipped groups in `result/rexy_sunbathe_mrx_Split.yaml`, see how there's only 4,
one for each segment in the split of `--split 3,5,6,7,3`. Try running it again,
this time changing the split to `--split 3,6,3`, how many skipped groups are
there now?

### The decodesave flag

The decodesave flag extracts every single metadata file and saves it to a folder,
relative to the partition and layout it was found in the mrx file.

The following command will split
`./testdata/rexy_sunbathe_mrx.mxf` into 3 folders labeled 0000StreamTC, 0001StreamBC and 0002StreamBC, where
each folder contains the metadata files in the order they were found in the file.

```console
./mrx-tool.exe decodesave --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx_contents/
```

Check out `result/rexy_sunbathe_mrx_contents/` see how the metadata has been split up
in the folder. Open `./testdata/rexy_sunbathe_mrx.mxf` in a tool like [MXF inspect](https://github.com/Myriadbits/MXFInspect),
see how the folder layout matches the mrx file layout.

Check [here](HELP.md#suggested-workflow-for-encoding) for more information about the naming of folders
and information about metadata streams.

The next flag (the encode flag) will go through the reverse process of making a file from this
and similar folder layouts.

### The encode flag

The encode flag groups together a collection of data file streams and
collates them as an mrx file. See the [help.md](HELP.md) for more detail on the encoding methods.

The example command will generate an mrx file, from the output of
the [the decodesave demo](#the-decodesave-flag), so please make sure you have run that
demo first.

```cmd
./mrx-tool.exe encode --input ./result/rexy_sunbathe_mrx_contents/ --output ./testdata/newrexy.mrx --framerate 24/1 
```

Checkout your newly generated mrx file at `./testdata/newrexy.mrx`,
try decoding it again see how the data hasn't changed from
the contents at `./result/rexy_sunbathe_mrx_contents/`.

### The split flag

The split flag is only for the `decode` command. It includes the user selected content packages into a skipped content package, shortening the overall length of the resulting yaml, but keeping the necessary information, such as total size. The skipped packages still effect the results of the statistic field.

The split element counts have the following effects:

- not using split does not skip any essence and extracts everything in the file.
- 1 element e.g. ```--split 4``` , this extracts the 4 metadata entries from the middle of the file.
- 2 elements e.g. ```--split 2,3``` , this extracts the first x amount of essence then the last y count of essence.
- 3+ elements e.g. ```--split 2,4,5,3``` , this extracts the first count of essence and then the last count of essence, with the remaining counts extracted from evenly interspersed positions in the file.

When odd numbers are used for extracting the middle essence, the count is split to be the "smaller half" from the mid point, then the "larger half". E.g. 3 at the 50th essence would take positions from 49 to 52, instead of 48 to 51.

## YAML Layout

The yaml contains an array of partitions and their essence information in the order they were found in the mrx file.

The partition section contains the following information

- Partition Type identifies the essence container. e.g. Header or Body
- HeaderLength is the length of the header in bytes and any metadata it may contain.
- EssenceByteCount is the total byte length of all the essence.
- ContentPackageCount is the number of individual content packages within the partition.
- IndexTable identifies if a index table is present in this partition showing some of the data contained within it.
- ContentPackages is an array of the content package found in the partition, in the order it was found in the partition.
Each content package is an essence array of any metadata it contains.
- Warning provides a string stating any potential issues within the essence, e.g. Essence is found in the header partition.
- skipped content is an object stating how many content packages were not included and their total byte count.
- ContentPackageStatistics contains the average, variance and standard deviation in the lengths of the content packages, as well as the longest and shortest package.

The essence array contains the following information in each element. It has the following fields.

- Key, this is the UL of the container.
- Symbol, this is the identifier of the essence type.
- Description, the description of the essence as found in the [smpte register][smpte-reg],
or auto generated information where the key was not identified.
- File Offset the offset in the file for the start of the data **NOT** the start of the essence container.
- length is the length of the essence data
- Type is the resolved container key if it can be found.
- TotalByteCount is the total count of the essence including the UL and BER encoded length Bytes.  

content packages with the key "00000000.00000000.00000000.00000000" are skipped content packages, these represent an array of content packages as a single item.

## Notes for developers

This section is to give some help
for including the go code in your own MRX encoders
and decoders.

### Internal Usage Example

The results yaml can be generated from any MRX byte stream.
ContentPackageCount is the go version of the [split flag](#the-split-flag)
and is an array of integers.

The following code will allow you to extract the mrx structure,
please make sure to change the names to be valid yaml and mrx files.

```go
package main

import (
    "os"
    "github.com/metarex-media/mrx-tool/decode"
)

func main() {
    // make sure to check all your errors
    yamltarget, _ := os.Create("my/target.yaml")
    contentPackageCount := []int{4,4}

    src , _ := os.Open("my/target.mrx") 

    err := decode.MRXStructureExtractor(src, yamltarget, contentPackageCount)

    if err != nil {
        panic(err)
    }

}

```

### Developing MRX writers

Check out the [example file for encoders](./encode/examples.go) for a full
example of a file writer.

The mrx encoder interface is broken into 3 sections:

- GetRoundTrip
- GetStreamInformation
- EssenceChannels

GetRoundTrip - this handles the [roundtrip file](./HELP.md#the-mrx-roundtrip-file)
that may be associated with the metadata. This is shown in the example code below.

```go
// ExampleFileStream contains the bare minimum
// to get a data stream saved as an MRX.
type ExampleFileStream struct {
    //  A dummy manifest foe examples
    FakeRoundTrip *manifest.Roundtrip

    Contents SingleStream
}


// GetRoundTrip returns the dummy roundtrip
func (st ExampleFileStream) GetRoundTrip() (*manifest.Roundtrip, error) {
    return st.FakeRoundTrip, nil
}

```

GetStreamInformation - this tells the writer, how many
channels there are and what the data type each channel will be.
This then lets the MRX encoder multiplex any relevant channels together.

In the example below only one set type of channel is being returned.

```go
// GetStreamInformation tells the mrx writer that their is one channel
// and that it is of type clocked text.
func (st ExampleFileStream) GetStreamInformation() (StreamInformation, error) {

    base := StreamInformation{ChannelCount: 1, EssenceKeys: []EssenceKey{TextFrame}}
    return base, nil
}
```

EssenceChannels - this handles the metadata channels
and sends the data through the channles to be handled by the writer.

In the example below only one channel is being written to.
When writing this it is imperative that all the channels you
send metadata are closed after they are finished writing to.
As this signals to the writer that the metadata streams have finished
and the file can be finished being written.

```go
// EssenceChannels is a simple essemce pipe that just puts the data straight through
func (st *ExampleFileStream) EssenceChannels(essChan chan *ChannelPackets) error {

    dataTrain := make(chan *DataCarriage, 10)
    mrxData := ChannelPackets{Packets: dataTrain}
    // send the single channel into the essence channel
    essChan <- &mrxData
    data := st.Contents

    max := len(data.MdStream)
    for i := 0; i < max; i++ {
        d, ok := <-data.MdStream

        if !ok {
            break
        }
        deref := d
        dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}

    }
    // close the channel to stop deadlocks
    close(dataTrain)
    return nil
}
```

[g1]:   https://go.dev/doc/install                "Golang Installation"
[smpte-reg]: https://registry.smpte-ra.org/apps/pages/published/ "SMPTE registers"
