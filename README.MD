# MRX Tool

MRX Tool is a command line tool, for all your mrx parsing needs. This tool is designed to help unravel the mysteries of an mrx file, with what it contains, how it contains it and any other questions you may have about a certain file.

## getting the MRX Tool

Please get the latest version for your operating system from the downloads page.

This tool is designed to be run from the command line

### The decode flag

The decode flag breaks down the selected mrx file into a [yaml file](#yaml-layout), detailing the labels of its contents and the overall file structure.

The --input and --output flags must be used for selecting the file and saving the output, it also contains an optional [--split flag](#split-function-usage), incase you want to reduce the yaml size.

Run the following commands as an example and check the difference between the two files that are generated.

```console
./mrxtool.exe decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx.yaml
./mrxtool.exe decode --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx2.yaml --split 3,5,6,7,3
```

### The decodesave flag

The decodesave flag extracts every bit of essence and saves it in a folder relative to the partition it was found in. The following command

```console
./mrxtool.exe decodesave --input ./testdata/rexy_sunbathe_mrx.mxf --output result/rexy_sunbathe_mrx_contents/
```

would generate three folders, each labeled body1, body2 and body3. Each folder would contain essence files sequentially ordered from essence0, in the order they are found in the file.

### The encode flag

The encode flag groups together a collection of data files and collates them as an mrx file. See the help.md for more detail on the encoding methods.

``` ./mrxtool.exe encode --input ./encode/testdata/testbase --output ./testdata/newrexy.mrx --framerate 24/1 ```

### split function usage

The split flag is only for the decode command. It compresses the user selected content packages into a skipped content package, shortening the overall length of the resulting yaml, but keeping the neccessary information, such as total size. The skipped packages still effect the results of the statistic field.

- not using split does not skip any essence and extracts everything in the file.
- 1 element e.g. ```--split 4``` , this extracts the essence from the middle of the file.
- 2 elements e.g. ```--split 2,3``` , this extracts the first x amount of essence then the last y count of essence.
- 3+ elements e.g. ```--split 2,4,5,3``` , this extracts the first count of essence and then the last count of essence, with the remaining counts extracted from evenly interspersed positions in the file.

When odd numbers are used for extracting the middle essence, the count is split to be the "smaller half" from the mid point, then the "larger half". E.g. 3 at the 50th essence would strart the count from 49 to 52.

## YAML Layout

The yaml contains an array of partitions and their essence information in the order they were found in the mrx file.

The partition section contains the following information

- Partition Type identifies the essence container. e.g. Header or Body
- HeaderLength is the length of the header and any metadata it may contain.
- EssenceByteCount is the total byte length of all the essence.
- ContentPackageCount is the number of individual content packages within the partition.
- IndexTable indetifies if a index table is present in this partition showing some of the data contained within it.
- ContentPackages is an array of the contentpackage found in the partition, in the order it was found in the partition. Each conent package is an essence array.
- Warning provides a string stating any potential issues within the essence.
- skipped content is an object stating how many content packages were not included and their total byte count.
- ContentPackageStatistics contains the average, variance and standard deviation in the lengths of the content packages, as well as the longest and shortest package.

The essence array contains the following information in each element. It has the following fields.

- Key, this is the UL of the container.
- Symbol, this is the identifier of the essence type.
- Description, the description of the essence as found in the smpte register, or auto generated information where the key was not identified.
- File Offset the offset in the file for the start of the data **NOT** the start of the essence container.
- length is the length of the essence data
- Type is the resolved container key if it can be found.
- TotalByteCount is the total count of the essence including the UL and BER encoded length Bytes.  

content packages with the key "00000000.00000000.00000000.00000000" are skipped content packages, these represent an array of content packages as a single item.

## Streaming data layout

The data is formatted in three distinct decoupling phases. These are designed to send the data in th eorder it is found in the reader.

Using an io.Reader interface the data is split into buffered data and fed into a channel.
This buffer channel is then split into KeyLengthValue (klv) packets, where each packet is a self contained klv.

The klv is then organised into the yaml layout as described earlier. Header metadata KLVs are discarded, as is the data as we are only intersted in the key and length of the data.

## Internal Usage Example

The results yaml can be generated from a local file or a byte stream, this is written to the writer the user provides.
ContentPackageCount is the maximum number of content packages to be displayed in the yaml per partition, the remaining information will be provided in the skip section of the yaml. In the following example each partition will only display up to four content packages, if there is only one package then one shall be displayed.

```go
package main

import (
    "os"
    "gitlab.com/mmTristan/mrxtool"
)

func main() {

    yamltarget, _ := os.Create("my/target.yaml")
    contentPackageCount := []int{4,4}
    err := mrxtool.ExtractEssence("my/target.mrx", yamltarget, contentPackageCount)

    if err != nil {
        panic(err)
    }

}

```
