# MRX Tool

MRX Tool is a utility tool and go library for interacting
with [Metarex](https://metarex.media/) MRX files.
It is used for encoding and decoding MRX files,
with a focus on improving the transparency of what makes an mrx file.
If you need help understanding mrx files, or want to make your own mrx files
and don't know where to start, then this is the tool for you.

## Contents

- [Overview](#overview)
- [Installation](#installation)
- [Demos](#demos)
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
- [extracting](#the-decode-flag) mrx files into the constituent metadata files
- [decoding](#the-decodesave-flag) the structural layout of an mrx file in json or yaml form
- [encoding](#the-encode-flag) metadata file(s) into a single mrx file

This repo also doubles up as a go library for encoding MRX files,
check out the [notes for developers](#notes-for-developers) for information
about the encoding process and its design. You can also checkout the go docs
for the encoding library.

Using this tool can help you to understand the contents
and specification of the mrx files.
Furthermore the mrx files this tool makes can be used as test vectors,
for testing your own mrx implementations.

## Installation

Please get the latest version for your operating system from the downloads page.

Or alternatively, download the repo and compile using the following command.
Make sure you have the latest version of Go from the [official golang
source][g1] installed. This is a pure go project with no
other dependencies.

```cmd
go build
```

This tool only runs on the command line.

Alternatively if you are utilising the library features then you
can install the repo with.

```cmd
go get github.com/metarex-media/mrx-tool
```

## Demos

There is a separate [demo repo](https://github.com/metarex-media/mrx-demo),
which gives an introduction and comprehensive walkthrough of the features of mxt-tool.

Alternatively the [tool functions](#mrx-tool-functions) also includes
demos of each command available.

## MRX Tool functions

The following commands are available, each section explains the command
and gives a brief demo of how to use it:

- [decoding](#the-decode-flag) the structural layout of an mrx file in yaml form
- [decoding](#the-decodesave-flag) mrx files into the metadata sub components
- [encoding](#the-encode-flag) metadata file(s) into a single mrx file

### The decode flag

The decode flag breaks down the selected mrx file structure into an file
detailing its layout and contents.
The produced yaml file has this [layout](#yaml-layout).

When running this command the following options are available.
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

The `decodesave` flag extracts every single metadata entry from the
mrx file and saves it to a folder.
The folder layout matches the layout of the mrx file it was extracted from.

The following command will split
`./testdata/rexy_sunbathe_mrx.mxf` into 3 folders labeled 0000StreamTC,
0001StreamBC and 0002StreamBC, where
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

The encode flag encodes folders of metadata files into a single MRX file.
See the [help.md](HELP.md) for more detail on the encoding methods and the
folder naming scheme.

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

The split flag is only for the `decode` command and it shortens the contents
of the output file by only including some of the metadata entries in
its summary.
It includes
the user selected content packages and skipped content package(s),
shortening the overall length of the resulting yaml, but keeping the necessary information,
such as total size. The skipped packages still effect the results of the statistic field.

The split element counts have the following effects:

- not using split does not skip any essence and extracts everything in the file.
- 1 element e.g. ```--split 4``` , this extracts the 4 metadata entries from the middle of the file.
- 2 elements e.g. ```--split 2,3``` , this extracts the first x amount of essence then the last y count of essence.
- 3+ elements e.g. ```--split 2,4,5,3``` , this extracts the first count of essence and then the last count of essence,
with the remaining counts extracted from evenly interspersed positions in the file.

When odd numbers are used for extracting the middle essence, the count is split to be the "smaller half" from the mid point, then the "larger half". E.g. 3 at the 50th essence would take positions from 49 to 52, instead of 48 to 51.

## YAML Layout

The YAML layout section details the structure and fields
of the data file produced by the [decode method.](#the-decode-flag)

The YAML contains an array of partitions and their essence information in the order they were found in the mrx file.

The partition section contains the following information

- `Partition Type` identifies the essence container. e.g. Header or Body
- `HeaderLength` is the length of the header in bytes and any metadata it may contain.
- `EssenceByteCount` is the total byte length of all the essence.
- `ContentPackageCount` is the number of individual content packages within the partition.
- `IndexTable` identifies if a index table is present in this partition showing some of the data contained within it.
- `ContentPackages` is an array of the content package found in the partition, in the order it was found in the partition.
Each content package is an essence array of any metadata it contains.
- `Warning` provides a string stating any potential issues within the essence, e.g. Essence is found in the header partition.
- `skipped content` is an object stating how many content packages were not included and their total byte count.
- `ContentPackageStatistics` contains the average, variance and standard deviation in the lengths of the content packages, as well as the longest and shortest package.

The essence array contains the following information in each element. It has the following fields.

- `Key`, this is the  universal label (UL) of the container.
- `Symbol`, this is the identifier of the essence type.
- `Description`, the description of the essence as found in the [smpte register][smpte-reg],
or auto generated information where the key was not identified.
- `File Offset`, the offset in the file for the start of the data **NOT** the start of the essence container.
- `length` is the length of the essence data
- `Type` is the resolved container key if it can be found.
- `TotalByteCount` is the total count of the essence including the UL and BER encoded length Bytes.  

content packages with the key "00000000.00000000.00000000.00000000" are skipped content packages, these represent an array of content packages as a single item.

## Notes for developers

This section gives examples and design information,
about including the go library code in your own MRX encoders
and decoders.

### Including decode functions

This section walks you through utilising the MRX decode functions
available as part of the library.

The MRX to yaml function is exported as `decode.MRXStructureExtractor`.
This handles all the structural decoding and saves the output straight to
the writer.

The following code example will allow you to extract the mrx structure,
please make sure to change the names to be valid yaml and mrx files.
The `ContentPackageCount` variable is the go version of the [split flag](#the-split-flag)
and is an array of integers.

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

### Developing MRX Encoders

This section walks you through using the MRX encode functions
available as part of the library. This library contains a
couple of out of the box
example MRX writers that are ready to go, as well as the options
for making your own encoders.

The first part covers developing you own encoder using the encoder interface.
Check out the [example file for encoders](./encode/examples.go) for a full
example of a file writer object. This writer handles all the MRX header
generation and file layout, as a user the only thing that needs to be
handled are the metadata streams into this file writer.

The workflow for creating an MRX writer is as follows:

```go
// 1. get the MRX writer object
mw = encode.NewMRXWriter()
// Or optionally include a set frame rate writer
// mw, err = encode.NewMRXWriterFR(encodeFrameRate)


// 2. update the Encoding object of the writer
mw.UpdateEncoder(writeMethod)

// 3. run the encoder
err = mw.Encode(w, &encode.MrxEncodeOptions{ManifestHistoryCount: encodeManifestCount, ConfigOverWrite: update})
```

The mrx object uses an encoder object to handle the metadata streams.
It uses the encoder interface which is broken into 3 methods:

- GetRoundTrip
- GetStreamInformation
- EssenceChannels

The following examples are excerpts from the
[single stream example encoder][encEx].
There is more information later on about the
[MRX encoder design](#mrx-encoder-design) and
the principles for making your own encoders.

**GetRoundTrip** - this handles the [roundtrip file](./HELP.md#the-mrx-roundtrip-file)
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

**GetStreamInformation** - this tells the writer, how many
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

**EssenceChannels** - this handles the metadata channels
and sends the data through the channels to be handled by the writer.

In the example below only one channel is being written to.
When writing this it is imperative that all the channels you
send metadata are closed after they are finished writing to.
As this signals to the writer that the metadata streams have finished
and the file can be finished being written.

```go
// EssenceChannels is a simple essence pipe that just puts the data straight through
func (st *ExampleFileStream) EssenceChannels(essChan chan *ChannelPackets) error {

    dataTrain := make(chan *DataCarriage, 10)
    defer close(dataTrain)
    mrxData := ChannelPackets{Packets: dataTrain}

    // send the single channel into the essence channel
    essChan <- &mrxData
    data := st.Contents

    max := len(data.MdStream)
    for i := 0; i < max; i++ {
        d, ok := <-data.MdStream

        if !ok {
            return fmt.Errorf("data channel unexpectedly closed")
        }
        deref := d
        dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}

    }
    // close the channel to stop deadlocks

    return nil
}
```

#### MRX encoder design

This section is about designing your own MRX encoders.
The encode library contains the methods for writing the MRX and
the focus of this section is passing the metadata to the writer.

There are two MRX encoder options currently available within the mrx-tool repo.
These are fully working intergrations that you can use.
The [folderscan code](./folderscan/folderscan.go), is used for the cli
and hoovering up locally saved files to be encoded as an MRX file. And the
[single stream example encoder][encEx], which
streams a single stream of metadata to
the MRX encoder.

These both stream the metadata to the encoder,
but in two different ways.

1. FolderScan runs as the encode function progresses,
we'll call this linear encoding.
2. Single stream runs concurrently before and during the mrx
encoding function, we'll call this stream encoding.

##### Linear encoding

Linear encoding runs as the encoder runs,
where the metadata extraction functions run dynamically through the
mrx file writing process.
In this section we will walk you through linear encoding, and
how it works within the folder scan library.

The encoder interface is broken into 3 methods
and runs in the following order when encoding files:

1. GetStreamInformation
2. GetRoundTrip
3. EssenceChannels

As the MRX encoder is setting up, it first calls  `GetStreamInformation`
This returns the metadata properties of each data stream, of what metadata type
the stream is and how many data streams there are in total. This is so it can
write the header partition of the MRX file and set up the data stream
for encoding.
Folder scan handles this
by scanning the target folder and every file within,
getting their essence types and aggregating the files for streaming. It
stores all this data for handling the streams later.

Next the MRX encoder then calls `GetRoundTrip`, this returns information
from previous MRX iterations of the metadata. Folderscan does this
by searching for the config.json in the parent folder it was given
and returning the json as the go object if it was found.

Finally the encoder calls `EssenceChannels` to send the metadata
to the writer. This handles a channel that each entry represents
a set of metadata, which then has a channel that sends the
packets of metadata.
When `EssenceChannels` is called for folder scan it,
searches for the files found when `GetStreamInformation` was called and
concurrently runs the extraction for each folder.
The extracted metadata is then sent down the relevant channel for each writer.

##### Stream encoding

Stream encoding is intended to run concurrently while
the mrx file is being encoded. This differs from the
linear encoding as it knows what is in the channels before
the encoding has begun. So instead of calculating the values
when the `GetRoundTrip` and `GetStreamInformation` functions are called,
it returns precalculated objects
from the stream initialisation.

In this example the stream always handles data of one type,
which is reflected in `GetSteamInformation` and the roundtrip
returns the user input.

```go
// GetStreamInformation tells the mrx writer that their is one channel
// and that it is of type clocked text.
func (st ExampleFileStream) GetStreamInformation() (StreamInformation, error) {

    base := StreamInformation{ChannelCount: 1, EssenceKeys: []EssenceKey{TextFrame}}
    return base, nil
}
```

Then the `EssenceChannels` handles the streams as they are concurrently running.
The following code is an example of the concurrent writing.

```go

    //send your data
    in := make(chan []byte, 10)
    go func(){
    // fil the channel with example data then close it
    for i := 0; i < 10; i++ {
        in <- []byte(`{"test":true}`)
    }
    close(in)
    } ()

    // run the writer here
    err := EncodeSingleDataStream(io.Discard, in, nil)

```

## Things to add - future work

The following is a list of things to be added to the library.

- The MRX unit test needs to be completed and integrated.

If you think we've missed something or have any requests then
please create an issue.

[g1]:   https://go.dev/doc/install                "Golang Installation"
[smpte-reg]: https://registry.smpte-ra.org/apps/pages/published/ "SMPTE registers"
[encEx]: ./encode/examples.go