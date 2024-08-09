# MRX Tool

MRX Tool is a utility tool and go library for interacting
with [Metarex](https://metarex.media/) MRX files.
It is used for encoding and decoding MRX files,
with a focus on improving the transparency of what makes an mrx file.
If you need help understanding mrx files, or want to make your own mrx files
and don't know where to start, then this is the tool for you.

[![Go Reference](https://pkg.go.dev/badge/github.com/metarex-media/mrx-tool.svg)](https://pkg.go.dev/github.com/metarex-media/mrx-tool)

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
- [Extra Tools to Visualise MRX files](#extra-tools-to-visualise-mrx-files)

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

For more technical information about
this tool and the MRX design check out the
[Help.md][hlp] file.

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


There is a separate [demo repo][mrx-demo],

which gives an introduction and comprehensive walkthrough of the features of mxt-tool.

Alternatively the [tool functions](#mrx-tool-functions) section also includes
demos of each command available.

There is also an [examples folder][ef] for the library functions of this repo.

## MRX Tool functions

The following commands are available, each section explains the command
and gives a brief demo of how to use it:

- [decoding](#the-decode-flag) the structural layout of an mrx file in yaml form
- [decoding](#the-decodesave-flag) mrx files into the metadata sub components
- [encoding](#the-encode-flag) metadata file(s) into a single mrx file

### The decode flag

The decode flag breaks down the selected mrx file structure into
a data file detailing its layout and contents.
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
./mrx-tool decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx.yaml
./mrx-tool decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx_Split.yaml --split 3,5,6,7,3
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
mrx file and saves each one as an individual file.
The folder layout matches the layout of the mrx file it was extracted from.

The following command will split
`./testdata/rexy_sunbathe_mrx.mxf` into 3 folders labeled 0000StreamTC,
0001StreamBC and 0002StreamBC, where
each folder contains the metadata files in the order they were found in the file.

```console
./mrx-tool decodesave --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx_contents/
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
See the [help.md][hlp] for more detail on the encoding methods and the
folder naming scheme.

The example command will generate an mrx file, from the output of
the [the decodesave demo](#the-decodesave-flag), so please make sure you have run that
demo first.

```cmd
./mrx-tool encode --input ./result/rexy_sunbathe_mrx_contents/ --output ./testdata/newrexy.mrx --framerate 24/1 
```

Checkout your newly generated mrx file at `./testdata/newrexy.mrx`,
try decoding it again see how the data hasn't changed from
the contents at `./result/rexy_sunbathe_mrx_contents/`.

### The split flag

The split flag is only for the `decode` command and it shortens the contents
of the output file, by only including some of the metadata entries in
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

When odd numbers are used for extracting the middle essence, the count is split to be the
"smaller half" from the mid point, then the "larger half". E.g. 3 at the 50th essence
would take positions from 49 to 52, instead of 48 to 51.

## YAML Layout

The YAML layout section details the structure and fields
of the data file produced by the [decode method.](#the-decode-flag)

The YAML contains an array of partitions and their essence information
in the order they were found in the mrx file.

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
- `ContentPackageStatistics` contains the average, variance and standard deviation in the lengths of the content
packages, as well as the longest and shortest package.

The essence array contains the following information in each element. It has the following fields.

- `Key`, this is the  universal label (UL) of the container.
- `Symbol`, this is the identifier of the essence type.
- `Description`, the description of the essence as found in the [smpte register][smpte-reg],
or auto generated information where the key was not identified.
- `File Offset`, the offset in the file for the start of the data **NOT** the start of the essence container.
- `length` is the length of the essence data
- `Type` is the resolved container key if it can be found.
- `TotalByteCount` is the total count of the essence including the UL and BER encoded length Bytes.  

content packages with the key "00000000.00000000.00000000.00000000" are skipped
content packages, these represent an array of content packages as a single item.

## Notes for developers

This section gives examples and design information,
about including the library code in your own MRX encoders
and decoders.

Checkout the [examples folder][ef] to see the working examples in action.

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
err = mw.Encode(w, &encode.MrxEncodeOptions{})
```

The mrx object uses an encoder object to handle the metadata streams.
It uses the encoder interface which is broken into 3 methods:

- GetRoundTrip
- GetStreamInformation
- EssenceChannels

The following examples are excerpts from the
[multiple stream example encoder][encEx].
There is more information later on about the
[MRX encoder design](#mrx-encoder-design) and
the principles for making your own encoders.

**GetRoundTrip** - this handles the [roundtrip file](./HELP.md#the-mrx-roundtrip-file)
that may be associated with the metadata. This is shown in the example code below.

```go
// ExampleFileStream contains the bare minimum
// to get multiple data streams saved as an MRX.
type ExampleMultipleStream struct {
    //  A dummy manifest foe examples
    RoundTrip  *manifest.RoundTrip
    StreamInfo *StreamInformation
    Contents   []SingleStream
}


// GetRoundTrip returns the roundtrip file
func (st ExampleMultipleStream) GetRoundTrip() (*manifest.RoundTrip, error) {
    return st.RoundTrip, nil
}
```

**GetStreamInformation** - this tells the writer, how many
channels there are and what the data type each channel will be.
This then lets the MRX encoder multiplex any relevant channels together.

In the example below the streams are based are on the input, with
an error check to ensure all the keys are correct.

```go
// GetStreamInformation tells the mrx writer the channel keys for the metadata.
// The number of keys is the number of channels.
func (st ExampleMultipleStream) GetStreamInformation() (StreamInformation, error) {

    if st.StreamInfo == nil {
        return StreamInformation{}, fmt.Errorf("no stream information found, ensure the stream is initialised")
    }

    return *st.StreamInfo, nil
}
```

**EssenceChannels** - this handles the metadata channels
and sends the data through the channels to be handled by the writer.

In the example below multiple channels are being written to.
When writing this it is imperative that all the channels you
send metadata are closed, after they are finished writing to.
As this signals to the writer that the metadata streams have finished
and the file can be finished being written.

```go
// EssenceChannels is a pipe that concurrently
// runs all the metadata streams at once.
func (st *ExampleMultipleStream) EssenceChannels(essChan chan *ChannelPackets) error {

    // use errs to handle errors while running concurrently
    errs, _ := errgroup.WithContext(context.Background())

    // initiate the klv stream

    for i, stream := range st.Contents {
        // set up the stream outside of the concurrent loop to preserve order
        pos := i
        dataTrain := make(chan *DataCarriage, 10)
        mrxData := ChannelPackets{Packets: dataTrain, OverViewData: manifest.GroupProperties{StreamID: pos}}
        essChan <- &mrxData

        errs.Go(func() error {
            // close the channel to stop deadlocks
            defer close(dataTrain)

            d, ok := <-stream.MdStream
            for ok {

                deref := d
                dataTrain <- &DataCarriage{Data: &deref, MetaData: &manifest.EssenceProperties{}}
                d, ok = <-stream.MdStream
            }

            return nil
        })
    }
    

    return errs.Wait()
}
```

#### MRX encoder design

This section is about designing your own MRX encoders.
The encode library contains the methods for writing the MRX and
the focus of this section is passing the metadata to the writer.
As the internals of the library handle the layout of the MRX file,
that us generated.

There are two MRX encoder options currently available within the mrx-tool repo.
These are fully working integrations that you can use.
The [folderscan code](./folderscan/folderscan.go), is used for the cli
and hoovering up locally saved files to be encoded as an MRX file. And the
[multiple stream example encoder][encEx], which
streams any amount of streams of metadata to
the MRX encoder.

These both stream the metadata to the encoder,
but in two different ways.

FolderScan runs as the encode function progresses.
This is because the encoder interface is broken into 3 methods
and runs in the following order when encoding files:

1. GetStreamInformation
2. GetRoundTrip
3. EssenceChannels

Folderscan saves the outputs from these functions and saves
them for later function calls.

The multiple stream runs concurrently before and during the mrx
encoding function. It is set up with an initialisation call that
saves all the information and starts writing the metadata. It is
not dependant on the MRX writer to start the metadata sending process.

## Extra Tools to Visualise MRX files

The following tools are also available to help get a greater
understanding of the contents of an MRX file. These are independently generated
mxf verification tools, showing that mrx, at its core is just an mxf file. That follows
the 20 year old standard.

- [MXF inspect](https://github.com/Myriadbits/MXFInspect) you can look at the physical layout of the file.
- [Reg-XML](https://registry.smpte-ra.org/apps/regxmldump/view/published/)
Gives more details here about the header information of an mrx file.

## Things to add - future work

The following is a list of things to be added to the library.

- The MRX unit test needs to be completed and integrated.

If you think we've missed something or have any requests then
please create an issue.

[g1]:   https://go.dev/doc/install                "Golang Installation"
[smpte-reg]: https://registry.smpte-ra.org/apps/pages/published/ "SMPTE registers"
[encEx]: ./encode/examples.go "example encoder file"
[hlp]: ./HELP.md "help.md"
[ef]: ./examples/ "examples folder"
[mrx-demo]: https://github.com/metarex-media/mrx-demo "MRX-demo"
