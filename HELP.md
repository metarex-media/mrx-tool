# MRX Tool

This is the technical documentation for MRX-tool, with
MRX design information and file design guides.
Please make sure you have read the [README][rm] and
are familiar with mrx-tool before reading this
documentation.

## Contents

- [Scope](#scope)
- [How to use](#how-to-use)
- [Suggested workflow for encoding](#suggested-workflow-for-encoding)
- [Metadata Timing](#metadata-timing)
- [The Mrx Roundtrip File](#the-mrx-roundtrip-file)
  - [Configuration](#configuration)
  - [The manifest](#the-manifest)
  - [Optional Parameters](#optional-parameters)
- [MRX Design Documentation](#mrx-design-documentation)
- [Metarex Glossary](#metarex-glossary)

## Scope

mrx-tool is a command line tool for containerising and uncontainerising
metadata, to and from a single Metadata Resource Express (MRX) file. This
command line tool will contain the means for simple encoding and decoding of mrx
files and not any methods for handling the metadata.

All metadata can be categorised using Metarex as one of four data types:

- Binary Clip Data
- Binary Frame Data
- Text Clip Data
- Text Frame Data

Frame data is clocked, this means it has explicit timing information generated in the mrx
file header. Clip wrapped data can contain embedded timing information
(clocked), or contain unclocked data, such as a schema.

This command line tool does not feature custom data types, e.g. data for private
use. However, it is still designed to unwrap them, but they will be flagged as
generic essence, instead of the private data type.

This command line is not designed for creating metadata to encode, or for
handling decoded metadata. It is only intended to encode and decode the metadata
into a file so it can be transported. What you do with the extracted metadata is
up to you.

This project is currently in development and may not be reflective of the final
MRX file design.

## How to Use

Check out the [readme](./README.MD) for running MRX tool.

## Suggested workflow for encoding

This section takes you through using mrx-tool for encoding groups of
metadata files as MRX files.

The recommended workflow for encoding mrx files using this tool, is to generate
your metadata channels into an ordered file system with format described below.
Then using the CLI, encode that file system as an mrx file, making sure to declare
the frame rates of the metadata in the [configuration file](#configuration).

When generating your data into the file system, for it to be encoded, the
following file system layout is to be used. The folder naming layout has a
header which identifies the channel with the naming format 0000Stream{{DataType}},
where each channel is to hold only one type of data,
please note a channel will be multiplexed with
other channels if these contain frame wrapped data. The channels must start as
0000Stream and increase incrementally, so they can be configured correctly.
The data files within the folder must follow the naming
sequence of 0000d, with no file extension. Where their numerical order in the
folder is order they will be generated in the file.

The dataType values and their meaning are:

- `TE` - Text Embedded data
- `TC` - Text clocked (frame wrapped) data
- `BE` - Binary Embedded data
- `BC` - Binary clocked (frame wrapped) data

So an example file system may look like:

- `0000StreamTC0000d`
- `0000StreamTC0001d`
- `0001StreamBE0000d`

or

- `0000StreamTC`
  - `0000d`
  - `0001d`
  - `0002d`
  - `0003d`
  - `0004d`
- `0001streamTE`
  - `0000d`

The mrx file can be validated by running the CLI again to extract the contents,
including the generated [manifest](#the-manifest), the extracted file system
should match the layout of the input filesystem. However due to the internal mrx
layout described [here](#mrx-design-documentation), the channels may be reordered
when being encoded and decoded again, this will occur if clip wrapped data is
placed before frame wrapped data.

for example this folder layout

- `0000StreamTC0000d`
- `0001StreamTE0000d`
- `0002StreamTC0000d`

would be reordered to if the cli was used to encode and decode and mrx file.

- `0000StreamTC0000d`
- `0001StreamTC0000d`
- `0002StreamTE0000d`

See how the embedded stream `StreamTE` was moved to the end
of the folders, because it is clip wrapped data.

## Metadata Timing

MRX is based off of the [Material Exchange Format (MXF)][MXFspec], which is
a video and audio file format, therefore it has inbuilt timing
for the metadata.

Metadata is currently measured in frames per second (fps) in an MRX file,
and the metadata within the file can be any fps. For frame wrapped data the default
frame rate is 24 fps if no configuration is provided or found.
The first frame wrapped metadata encountered in
the encoding process sets the frame rate for the rest of the frame wrapped data
and the mrx file.
This is because of the multiplexing of the metadata in MRX file, multiplexing
is the interlacing of all frame wrapped data together, so instead of being
a stream of x data and then a stream of y data, its a series of groups of x and y data
at each timing step.

For example if the first
frame wrapped data source has a timing of 24fps and the next data
is 48fps, then there will be one file of source 1 (24fps) for
every 2 files from source 2 (48fps) saved in the MRX file.

The data stream in the file would be
multiplexed together and look like so

- Source 1 (24fps)
- Source 2 (48fps)
- Source 2 (48fps)
- Source 1 (24fps)
- Source 2 (48fps)
- Source 2 (48fps)
- Source 1 (24fps)
- Source 2 (48fps)
- Source 2 (48fps)

It is good practice to ensure the metadata timings are integer multiples of each other,
to avoid timing and multiplexing issues. As you can't have half a metadata in a stream!

## The Mrx Roundtrip File

The final partition of the mrx contains the "RoundTrip file", which contains
the history of the generated data and
the configuration settings of the metadata channels.

The layout of the json is two top level fields of the Manifest and Configuration

```json
{

    "Configuration" : {"json data"},
    "Manifest" : {"json data"}
}
```

When this extracted by the command
line from an MRX it is saved as `config.json` in the parent folder.
It is also searched for by the encoder.

### Configuration

The configuration contains all the information about how
the file and its metadata was made.

The configuration contains the MRX file version, the default channel properties
and the stream properties of the individual channels. The default stream
properties and stream properties contain the same fields, so that substitutions
can be easily made. As the default properties are used if no stream properties
are declared.

These fields are the:

- `FrameRate` - this is in the form "x/y" or "static" and a label of the
data. "static" does not need to be declared, as it is not used internally for identifying
embedded data, but may be useful for reading over
the configuration to understand the metadata channel properties.
- `NameSpace` this identifies the namespace of the metadata.
- `Type` a description of the metadata.

An example configuration is below.

```json
 "Configuration" : {
        "MrxVersion": "pre alpha",
        "DefaultStreamProperties": {
            "FrameRate" : "24/1",
            "Type" : "some data to track",
            "NameSpace": "https://metarex.media/reg/MRX.123.456.789.def"
        },
        "StreamProperties" : {
            "1":{
                "FrameRate" : "24/1",
                "Type" : "CameraComponent"
            },
            "2": {
                "FrameRate"  : "static",
                "Type" : "Camera Schema"
            }
        }
 }
```

### The manifest

The manifest logs the history of the metadata, including
previous MRX iterations and previous manifests.

As the mrx file is encoded, an extra metadata component is generated in the file
called the manifest. The manifest logs the sha256 hash and other optional
metadata about the individual pieces of metadata in the mrx file. The order of
the metadata in the manifest matches the order it is found in the mrx file.

The manifest layout contains several nested layers of metadata, for the overall
file, the individual metadata channels and the individual metadata files in the
channel. The top segment of the manifest contains the metadata of the mrx file
and the tool that made the manifest. Then the channel field, which is an array
of the individual channels, these share a "Common Data Properties" key that
contains metadata for that channel, such as the stream ID. Each channel contains
the essence array, which represents each metadata item in the channel. An
example manifest is given below.

```json
{
  "Hash": "db9da06cb619f3884d533c9e6cfd9bf8335f19f34bdbd948d2b4bc67e8dbe945",
  "DataOrigin": "C:\\example\\location\\0000frameText",
  "EditDate": "2023-06-15 14:14:15.4862202 +0100 BST",
  "Extra User Metadata" : {
      "An example" : "of the user data to be added"
  }
}
```

### Optional Parameters

When encoding the file, there are optional parameters to tune the manifest.

Manifest History - if the mrx file is generated from metadata with a pre existing
manifest, then it will be added on to the history metadata field of the new
manifest. The user can limit the number of manifests in the history field so
that the manifest does not contain redundant information.

## MRX Design Documentation

This section goes over the design for the MRX file and the similarities
and differences it has from MXF.

The overall design utilises a few current mxf methods and standards for wrapping
data, with some new additions being used when these do not cover our use cases.
All MRX files have been generated following the op1a operational pattern.

The frame wrapped data is multiplexed into a mxf stream, which shares the same
base timing for all the frame wrapped data. This is to be wrapped with left
aligned grouping, where the frame rate content is declared first. As in SMPTE
379-1 for generic containers,

The essence data keys cover:

1. The type of metadata, frame wrapped etc.
2. The essence count in that content package, e.g. third frame wrapped data
   type.
3. The count of that essence type (up to 127) in a content package.

The key frame rate essence is produced first, giving a single content package,
with the rest of the content packages trailing. There may be several keys of
essence in a single content package. A single frame (at the frame rate set by
the the first essence) is encompassed in a single content package, which may
contain several essence keys, until a new key frame essence is encountered. The
essence keys for the frame wrapped data, follow the generic container pattern.

Each essence key one has an essence count, at the 14th byte, with a count from 1
to 127 the essence count is the number of essence items in the content package.
The 16th byte is the element number, this is a unique value amongst essence keys
of the same type. The element number is the incremental count of this element
type, if three frame wrapped data elements are present then they will have
element numbers of 00, 01 and 02 respectively.

The IXSD essence key from does not follow this format. The 14th byte is the
element number and the 16th byte is the element type. This is different from the
generic partition key and the rdd 47 key values in their documents.
In the MRX specification we use the 14th byte as
the essence count and the 16th byte as the element number,
to keep conformity across Metarex.

Clip wrapped data (binary and text) follows the methods layed out in RP2057 and rdd 47, where text
based documents are stored in generic partitions. Each generic partition has an
incremental stream id, they are placed immediately before the manifest
partition. Then the footer partition follows the manifest partition.

The essence keys for generic stream partitions follow RP2057 and ST 410, with
the addition of the use of the 13th byte in the key. This is to flag if the data
is binary or text based where the 1st bit of the 13byte equals 1 for binary
data, it remains 0 for text data.

The timing rules are as follows:

- All clip wrapped data follows the timing method in rdd 47, where they are all
  linked to the same static track in the upper most file package. Even if they
  have embedded timing information.
- The first clocked channel is the frame rate for the rest of the data, and this
  is used for the output timeline in the material package.

## Metarex Glossary

- Channel - an instance of metadata with a single mrxId, channels of metadata may
be multiplexed into a single mrx stream.
- ChannelID - UMID + SID  + UID , the
UID in this case is the essence key within the stream.
- Channel PositionID - channelID + (frame * editrate), the ID of a frame of
metadata within a Channel, this will relate to a content package InstanceID -
PostionID + (if contentPackage > 1) sub frame edit rate, the ID of a specific
file within a frame, If the content package is a single item then the positionID
and ChannelID are the same.

[rm]: ./README.MD
[mxfspec]: https://pub.smpte.org/doc/377/
