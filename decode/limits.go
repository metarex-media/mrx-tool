package decode

import "math"

func contentPackageLimiter(allKeys []contentPackage, containers []container, contentPackageLimit []int) []container {

	total := 0
	for _, contentPackageLength := range contentPackageLimit {
		total += contentPackageLength
	}

	// get the total count
	keyCount := len(allKeys)

	if total < len(allKeys) {

		// get the array positions here
		groups := groupSplit(contentPackageLimit, keyCount)

		// if empty return all the data
		if len(groups) == 0 {
			return containers
		}
		/*	var groups []int
			switch len(contentPackageLimit) {
			case 0:

				// return the unfiltered containers
				return containers
			case 1:
				//return an average of the middles
				width := contentPackageLimit[0]
				startPoint := keyCount / 2 // 10/11 both go to 5
				essWidthMin := width / 2   // 3 goes to instead of 1.5
				essWidthMax := int(math.Round(float64(width) / 2))

				groups = []int{startPoint - essWidthMin, startPoint + essWidthMax}
			case 2:
				width := contentPackageLimit[0]
				lastwidth := contentPackageLimit[1]
				groups = []int{0, width, keyCount - lastwidth, keyCount} // return [:first] and [last:]
			default:

				width := contentPackageLimit[0]
				lastwidth := contentPackageLimit[len(contentPackageLimit)-1]
				groups = []int{0, width}

				for i, limits := range contentPackageLimit[1 : len(contentPackageLimit)-1] {

					// position is i+1 / fraction count
					// the denominator is the count - 1 as the start and ends total one position
					mid := ((i + 1) * len(allKeys)) / (len(contentPackageLimit) - 1)

					essWidthMin := limits / 2 // 3 goes to instead of 1.5
					essWidthMax := int(math.Round(float64(limits) / 2))
					position := []int{mid - essWidthMin, mid + essWidthMax}
					groups = append(groups, position...)

				}
				groups = append(groups, []int{keyCount - lastwidth, keyCount}...)

				// return [:first] and [last:] and the middle formula
			}*/

		//	fmt.Println(groups, contentPackageLimit)
		for position := 0; position < len(groups); position += 2 {
			start := groups[position]
			end := groups[position+1]
			for i := start; i < end; i++ {
				//	fmt.Println(start, end)
				allKeys[i].keep = true

			}

		}

	} else {
		// return the unfiltered containers
		return containers
	}

	// variables to mark the pseudo position within the container
	// and how many essence packs may have been skipped
	partition := 0
	container := 0
	skipcount := 0
	skipCumlative := 0

	count := 0
	for _, key := range allKeys {

		// check if the key should be kept
		// incrementing the skip count and total counts
		if !key.keep {
			skipcount += key.ContentPackageLength
			skipCumlative++
		} else if key.keep && skipcount != 0 {
			// if previous items have been skipped
			// then generate a skip key
			allKeys[count-1] = skip(skipcount, skipCumlative)

			skipcount = 0
			skipCumlative = 0
		}

		// when the container changes
		// check if skips are in play
		// reset positions back to 0
		if container == len(containers[partition].ContentPackages)-1 || len(containers[partition].ContentPackages) == 0 {

			if skipcount != 0 && len(containers[partition].ContentPackages) != 0 {

				allKeys[count-1] = skip(skipcount, skipCumlative)

				skipcount = 0
				skipCumlative = 0
			}

			container = 0
			partition++

		}

		// always move the total key position along
		// and the container withing a content package
		count++
		container++

	}

	i := 0
	for partition, container := range containers {
		// generate a new content package for each content package
		// based on the skiping method
		var newContentPackages []contentPackage
		for range container.ContentPackages {
			if allKeys[i].keep {

				newContentPackages = append(newContentPackages, allKeys[i])
			}
			i++
		}
		//fmt.Println(newContentPackages)
		containers[partition].ContentPackages = newContentPackages
	}

	return containers

}

func skip(count, skipCumlative int) contentPackage {
	return contentPackage{ContentPackageLength: count,
		ContentPackage: []keyLength{{Key: "00000000.00000000.00000000.00000000", Description: "A collection of skipped content packages", TotalByteCount: count, TotalContainerCount: skipCumlative}}, keep: true}
}

func groupSplit(contentPackageLimit []int, keyCount int) []int {

	var groups []int
	switch len(contentPackageLimit) {
	case 0:

		// return the unfiltered containers
		return groups
	case 1:
		//return an average of the middles
		width := contentPackageLimit[0]
		startPoint := keyCount / 2 // 10/11 both go to 5
		essWidthMin := width / 2   // 3 goes to instead of 1.5
		essWidthMax := int(math.Round(float64(width) / 2))

		groups = []int{startPoint - essWidthMin, startPoint + essWidthMax}
	case 2:
		width := contentPackageLimit[0]
		lastwidth := contentPackageLimit[1]
		groups = []int{0, width, keyCount - lastwidth, keyCount} // return [:first] and [last:]
	default:

		width := contentPackageLimit[0]
		lastwidth := contentPackageLimit[len(contentPackageLimit)-1]
		groups = []int{0, width}

		for i, limits := range contentPackageLimit[1 : len(contentPackageLimit)-1] {

			// position is i+1 / fraction count
			// the denominator is the count - 1 as the start and ends total one position
			mid := ((i + 1) * keyCount) / (len(contentPackageLimit) - 1)

			essWidthMin := limits / 2 // 3 goes to instead of 1.5
			essWidthMax := int(math.Round(float64(limits) / 2))
			position := []int{mid - essWidthMin, mid + essWidthMax}
			groups = append(groups, position...)

		}
		groups = append(groups, []int{keyCount - lastwidth, keyCount}...)

		// return [:first] and [last:] and the middle formula
	}

	return groups
}
