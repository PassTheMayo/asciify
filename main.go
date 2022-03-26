package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

var (
	ErrNoInput      = errors.New("missing input image argument")
	ChararacterSets = map[string]string{
		"ascii": ".'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$",
	}
)

type Options struct {
	Verbose bool    `short:"V" long:"verbose" description:"Prints additional debug information"`
	Output  string  `short:"o" long:"out" description:"The file to write the output to"`
	Resize  string  `short:"r" long:"resize" description:"Resize the image to specific dimensions"`
	Charset string  `short:"c" long:"charset" description:"The character set to use for the output" default:"ascii"`
	Scale   float64 `short:"s" long:"scale" description:"Scales image and preserves aspect ratio" default:"0"`
}

func luminance(color color.Color) float64 {
	r, g, b, _ := color.RGBA()

	red := float64(r) / math.MaxUint16
	green := float64(g) / math.MaxUint16
	blue := float64(b) / math.MaxUint16

	return float64(0.299*float64(red) + 0.587*float64(green) + 0.114*float64(blue))
}

func resize(img image.Image, width, height int) image.Image {
	output := image.NewNRGBA(image.Rect(0, 0, width, height))
	inputBounds := img.Bounds().Size()

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			ix := int((float64(x) / float64(width)) * float64(inputBounds.X))
			iy := int((float64(y) / float64(height)) * float64(inputBounds.Y))

			output.Set(x, y, img.At(ix, iy))
		}
	}

	return output
}

func parseResize(value string, img image.Image) (int, int, error) {
	if len(value) < 1 {
		size := img.Bounds().Size()

		return size.X, size.Y, nil
	}

	split := strings.SplitN(value, "x", 2)

	if len(split) < 2 {
		return 0, 0, fmt.Errorf("invalid resize value: %s", value)
	}

	width, err := strconv.ParseUint(split[0], 10, 32)

	if err != nil {
		return 0, 0, err
	}

	height, err := strconv.ParseUint(split[1], 10, 32)

	if err != nil {
		return 0, 0, err
	}

	return int(width), int(height), nil
}

func main() {
	opts := &Options{}

	args, err := flags.Parse(opts)

	if err != nil {
		if flags.WroteHelp(err) {
			return
		}

		panic(err)
	}

	if len(args) < 1 {
		panic(ErrNoInput)
	}

	if !strings.HasSuffix(args[0], ".png") && !strings.HasSuffix(args[0], ".jpg") && !strings.HasSuffix(args[0], ".jpeg") {
		panic(fmt.Errorf("unknown image format: %s", args[0]))
	}

	charset, ok := ChararacterSets[opts.Charset]

	if !ok {
		panic(fmt.Errorf("unknown character set: %s", opts.Charset))
	}

	if opts.Verbose {
		fmt.Printf("VERBOSE: Found character set '%s' (%d characters)\n", opts.Charset, len(charset))
	}

	f, err := os.Open(args[0])

	if err != nil {
		panic(err)
	}

	defer f.Close()

	if opts.Verbose {
		fmt.Printf("VERBOSE: Opened input image '%s'\n", args[0])
	}

	var img image.Image = nil

	if strings.HasSuffix(args[0], ".png") {
		img, err = png.Decode(f)

		if err != nil {
			panic(err)
		}
	} else {
		img, err = jpeg.Decode(f)

		if err != nil {
			panic(err)
		}
	}

	if opts.Verbose {
		fmt.Println("VERBOSE: Successfully parsed input image")
	}

	ow, oh, err := parseResize(opts.Resize, img)

	if err != nil {
		panic(err)
	}

	if opts.Scale != 0 {
		size := img.Bounds().Size()

		ow = int(float64(size.X) * opts.Scale)
		oh = int(float64(size.Y) * opts.Scale)
	}

	processedImg := resize(img, ow, oh)

	if opts.Verbose {
		fmt.Printf("VERBOSE: Resized image from %s to %s\n", img.Bounds().Size(), processedImg.Bounds().Size())
	}

	result := &bytes.Buffer{}

	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			lum := luminance(processedImg.At(x, y))
			char := charset[int(float64(len(charset))*lum)]

			if _, err = result.WriteString(fmt.Sprintf("%c", char)); err != nil {
				panic(err)
			}
		}

		if y+1 != oh {
			if _, err = result.WriteString("\n"); err != nil {
				panic(err)
			}
		}
	}

	if len(opts.Output) > 0 {
		outFile := args[0] + ".txt"

		if len(opts.Output) > 0 {
			outFile = opts.Output
		}

		if err = ioutil.WriteFile(outFile, result.Bytes(), 0777); err != nil {
			panic(err)
		}

		if opts.Verbose {
			fmt.Printf("VERBOSE: Successfully wrote output to '%s'\n", outFile)
		}

		return
	}

	fmt.Println(result.String())
}
