package tool

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"

	"github.com/disintegration/imaging"
)

// Grayscale converts the image to grayscale.
func Grayscale(img image.Image) image.Image {
	return imaging.Grayscale(img)
}

// Erode performs erosion using a specified kernel size.
func Erode(img image.Image, kernelSize int) image.Image {
	bounds := img.Bounds()
	eroded := imaging.Clone(img)

	radius := kernelSize / 2
	for y := bounds.Min.Y + radius; y < bounds.Max.Y-radius; y++ {
		for x := bounds.Min.X + radius; x < bounds.Max.X-radius; x++ {
			min := color.Gray{Y: 255}
			for ky := -radius; ky <= radius; ky++ {
				for kx := -radius; kx <= radius; kx++ {
					px := color.GrayModel.Convert(img.At(x+kx, y+ky)).(color.Gray)
					if px.Y < min.Y {
						min = px
					}
				}
			}
			eroded.Set(x, y, min)
		}
	}
	return eroded
}

// Threshold applies binary thresholding to binarize the image.
func Threshold(img image.Image, threshold, maxVal float64) image.Image {
	bounds := img.Bounds()
	binary := imaging.Clone(img)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			px := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			if float64(px.Y) > threshold {
				binary.Set(x, y, color.Gray{Y: uint8(maxVal)})
			} else {
				binary.Set(x, y, color.Gray{Y: 0})
			}
		}
	}
	return binary
}

// RemoveSmallRegions removes connected components smaller than the specified size.
func RemoveSmallRegions(img image.Image, minSize int) image.Image {
	bounds := img.Bounds()
	labels := make([][]int, bounds.Dy()) // Labels for connected components
	for i := range labels {
		labels[i] = make([]int, bounds.Dx())
	}

	label := 1
	area := make(map[int]int)
	var floodFill func(x, y int)

	// Flood fill to label connected components
	floodFill = func(x, y int) {
		if x < 0 || y < 0 || x >= bounds.Dx() || y >= bounds.Dy() || labels[y][x] > 0 {
			return
		}
		px := color.GrayModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray)
		if px.Y > 0 { // Skip non-black pixels
			return
		}
		labels[y][x] = label
		area[label]++
		floodFill(x+1, y)
		floodFill(x-1, y)
		floodFill(x, y+1)
		floodFill(x, y-1)
	}

	// Label all black regions
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if labels[y][x] == 0 {
				px := color.GrayModel.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray)
				if px.Y == 0 { // Black pixel
					floodFill(x, y)
					label++
				}
			}
		}
	}

	// Remove regions smaller than minSize
	output := imaging.Clone(img)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if labels[y][x] > 0 && area[labels[y][x]] < minSize {
				output.Set(bounds.Min.X+x, bounds.Min.Y+y, color.Gray{Y: 255}) // Set to white
			}
		}
	}

	return output
}

// RemoveSmallWhiteRegions removes white regions smaller than the specified width and height.
func RemoveSmallWhiteRegions(img image.Image, minWidth, minHeight int) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	labels := make([][]int, height)
	for i := range labels {
		labels[i] = make([]int, width)
	}

	label := 1
	regionSizes := make(map[int]int) // 紀錄區域大小
	var floodFill func(int, int, int, *int)

	// Flood fill 用來標記白色區域並計算大小
	floodFill = func(x, y, label int, size *int) {
		if x < 0 || y < 0 || x >= width || y >= height {
			return
		}
		if labels[y][x] > 0 { // 已標記
			return
		}

		px := img.At(bounds.Min.X+x, bounds.Min.Y+y)
		gray, _, _, _ := px.RGBA()
		if gray>>8 < 200 { // 如果不是白色 (灰階閾值 200)
			return
		}

		labels[y][x] = label
		*size++
		floodFill(x+1, y, label, size)
		floodFill(x-1, y, label, size)
		floodFill(x, y+1, label, size)
		floodFill(x, y-1, label, size)
	}

	// 逐像素標記區域
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if labels[y][x] == 0 {
				size := 0
				floodFill(x, y, label, &size)
				if size > 0 {
					regionSizes[label] = size
					label++
				}
			}
		}
	}

	// 過濾小於 minWidth * minHeight 的區域
	output := image.NewGray(bounds)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if regionSizes[labels[y][x]] < minWidth*minHeight {
				output.Set(bounds.Min.X+x, bounds.Min.Y+y, color.Gray{Y: 0}) // 設為黑色
			} else {
				output.Set(bounds.Min.X+x, bounds.Min.Y+y, img.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
	}

	return output
}

// Dilate performs dilation using a specified kernel size.
func Dilate(img image.Image, kernelSize int) image.Image {
	bounds := img.Bounds()
	dilated := imaging.Clone(img)

	radius := kernelSize / 2
	for y := bounds.Min.Y + radius; y < bounds.Max.Y-radius; y++ {
		for x := bounds.Min.X + radius; x < bounds.Max.X-radius; x++ {
			max := color.Gray{Y: 0}
			for ky := -radius; ky <= radius; ky++ {
				for kx := -radius; kx <= radius; kx++ {
					px := color.GrayModel.Convert(img.At(x+kx, y+ky)).(color.Gray)
					if px.Y > max.Y {
						max = px
					}
				}
			}
			dilated.Set(x, y, max)
		}
	}
	return dilated
}

// ProcessImage processes the input image and applies the given steps.
func ProcessImage(inputPath, outputPath string) error {
	// Step 1: Load image
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return err
	}

	// Step 2: Convert to grayscale
	grayImg := Grayscale(img)

	// Step 3: Apply erosion
	erodedImg := Erode(grayImg, 3)

	// Step 4: Apply binary thresholding after erosion
	binarizedAfterErosion := Threshold(erodedImg, 190, 255)

	// Step 5: Remove small noise regions
	denoisedImg := RemoveSmallRegions(binarizedAfterErosion, 30)

	// Step 6: Enhance contrast
	contrastImg := imaging.AdjustContrast(denoisedImg, 60)

	// Step 7: Remove small white regions
	cleanedImg := RemoveSmallWhiteRegions(contrastImg, 2, 2)

	// Step 8: Apply dilation to make characters thicker
	finalImg := Dilate(cleanedImg, 3)

	// Step 7: Save the final output image
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, finalImg, nil)
}
