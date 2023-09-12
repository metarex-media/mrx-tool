package essence

import "fmt"

const (
	prefix = "urn:smpte:ul:"
)

// ExtractEssenceType returns the essence information associated with a essence Key,
// if it a matching key found.
func ExtractEssenceType(UL []byte, matches map[string]EssenceInformation, pos *int) EssenceInformation {
	//prefix := "urn:smpte:ul:"

	if ess, ok := EssenceLookUp[prefix+fullNameTwo(UL)]; ok {
		return ess
	}

	if ess, ok := EssenceLookUp[prefix+fullNameOne(UL)]; ok {
		return ess
	}

	if ess, ok := EssenceLookUp[prefix+FullName(UL)]; ok {
		return ess
	}

	return unknownEssence(UL, matches, pos)
}

func unknownEssence(UL []byte, matches map[string]EssenceInformation, pos *int) EssenceInformation {

	if ess, ok := matches[string(UL)]; ok {
		return ess

	} else {
		sym := fmt.Sprintf("SystemItemTBD%v", *pos)

		newEss := EssenceInformation{Symbol: sym, UL: prefix + fullNameOne(UL)}
		matches[string(UL)] = newEss
		*pos++

		return newEss
	}

}

func fullNameTwo(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x7f%02x7f",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[14])
}

func fullNameOne(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x7f",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14])
}

func FullName(namebytes []byte) string {

	if len(namebytes) != 16 {
		return ""
	}

	return fmt.Sprintf("%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x.%02x%02x%02x%02x",
		namebytes[0], namebytes[1], namebytes[2], namebytes[3], namebytes[4], namebytes[5], namebytes[6], namebytes[7],
		namebytes[8], namebytes[9], namebytes[10], namebytes[11], namebytes[12], namebytes[13], namebytes[14], namebytes[15])
}
