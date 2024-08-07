package manifest

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed jsonschema/manifest_Schema.json
var ManifestSchema []byte

//go:embed jsonschema/configuration_Schema.json
var ConfigSchema []byte

// PreviousManifest updates the manifest with the previous manifest.
func PreviousManifest(manifest []byte) ([]TaggedManifest, error) {

	var oldManifest TaggedManifest
	err := json.Unmarshal(manifest, &oldManifest)
	if err != nil {
		return nil, err
	}

	history := oldManifest.History
	// self regulate previous to be 0
	// to prevent a very convulted nested json
	oldManifest.History = nil

	return append([]TaggedManifest{oldManifest}, history...), nil

}

// ManifestValidator checks that the mainfest is valid against the manifest schema.
// The verbose mode gives a full list of the errors, which may be a large string
func ManifestValidator(manifest []byte, verbose bool) error {
	schemaLoader := gojsonschema.NewBytesLoader(ManifestSchema)
	documentLoader := gojsonschema.NewBytesLoader(manifest)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if result.Valid() {
		return nil
	} else {
		errString := "The document is not valid. "

		if verbose {
			errString += "See errors :\n"
			for _, desc := range result.Errors() {
				errString += fmt.Sprintf("- %s\n", desc)
			}
		}
		errString += "\n"

		return fmt.Errorf(errString)
	}
	// anifest validator will just wrap all that schema info
}

// RoundTrip is the json structure for
// for the configuration and manifest.
type RoundTrip struct {
	Config   Configuration `json:"Configuration,omitempty"`
	Manifest Manifest      `json:"Manifest,omitempty"`
}

// Configuration is the configuration for the global
// file and each data stream.
type Configuration struct {
	Version string           `json:"MRXVersion,omitempty"`
	Default StreamProperties `json:"DefaultStreamProperties,omitempty"`

	StreamProperties map[int]StreamProperties `json:"StreamProperties,omitempty"`
}

// StreamProperties are the individual properties
// of the metadata
type StreamProperties struct {
	StreamType string `json:"Type,omitempty"`
	FrameRate  string `json:"FrameRate,omitempty"`
	NameSpace  string `json:"NameSpace,omitempty"`
}

// add this to the main mrx writer body
type Manifest struct {
	UMID    string `json:"UMID,omitempty"`                 // UMID of the mrx file
	Version string `json:"Mrx Manifest Version,omitempty"` // what mainfest version was this generated to
	MRXTool string // MRXTool if the program that generated ut
	// An array of the partitions and their contents
	DataStreams []Overview `json:"Data Streams,omitempty"`
	// Only the highest Manifest shall have the previous section
	// Manifests in the previous array shall keep the array open
	History []TaggedManifest `json:"History,omitempty" yaml:"History,omitempty"`
}

// TaggedManifest is the same as a Manifest,
// with addition of the date the manifest was last edited.
type TaggedManifest struct {
	Date string `json:"SnapShot Date,omitempty"`
	Manifest
}

type Overview struct {
	// give any metadata more localised metadata here
	// have the list of properties here
	Common  GroupProperties `json:"Common Data Properties,omitempty"`
	Essence []EssenceProperties
}

type GroupProperties struct {
	StreamID          int    `json:"StreamID,omitempty"`
	StreamType        string `json:"StreamType,omitempty"`
	StreamContentType string `json:"StreamContentType,omitempty"`
	// DataOriginBasePAth
	// Maybe another bit of data
	CustomMeta any `json:"Extra Group Metadata,omitempty" yaml:"Extra Group Metadata,omitempty"`
}

type EssenceProperties struct {
	Hash       string `json:"Hash" yaml:"Hash,omitempty"`                       // Notoptional
	DataOrigin string `json:"DataOrigin,omitempty" yaml:"DataOrigin,omitempty"` // optional as not everything is available from an os.Stat
	EditDate   string `json:"EditDate,omitempty" yaml:"EditDate,omitempty"`     // optional
	CustomMeta any    `json:"Extra User Metadata,omitempty" yaml:"Extra User Metadata,omitempty"`
}
