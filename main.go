package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chai2010/webp"
)

const MaxImageSize = 1023 * 1024 // 1023 KB

func die(err error) {
	if err != nil {
		panic(err)
	}
}

/////////////////////////////
// Compression
/////////////////////////////

func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func zlibDecompress(data []byte) []byte {
	r, err := zlib.NewReader(bytes.NewReader(data))
	die(err)
	defer r.Close()
	out, err := io.ReadAll(r)
	die(err)
	return out
}

/////////////////////////////
// ENCODE
/////////////////////////////

func encode(input, prefix string) {

	raw, err := os.ReadFile(input)
	die(err)

	fmt.Println("Compressing input...")
	data := zlibCompress(raw)

	offset := 0
	chunkIndex := 0

	for offset < len(data) {

		// start optimistic
		maxTry := len(data) - offset
		if maxTry > 900_000 {
			maxTry = 900_000
		}

		var finalPayload []byte
		var finalImage []byte

		for try := maxTry; try > 1024; try -= 4096 {

			payload := data[offset : offset+try]

			pixels := int(math.Ceil(float64(len(payload)) / 3))
			side := int(math.Ceil(math.Sqrt(float64(pixels))))

			rgb := make([]byte, side*side*3)
			copy(rgb, payload)

			img := image.NewRGBA(image.Rect(0, 0, side, side))
			i := 0
			for y := 0; y < side; y++ {
				for x := 0; x < side; x++ {
					img.Set(x, y, color.RGBA{
						R: rgb[i],
						G: rgb[i+1],
						B: rgb[i+2],
						A: 255,
					})
					i += 3
				}
			}

			var buf bytes.Buffer
			err := webp.Encode(&buf, img, &webp.Options{
				Lossless: true,
				Quality:  100,
			})
			die(err)

			if buf.Len() <= MaxImageSize {
				finalPayload = payload
				finalImage = buf.Bytes()
				break
			}
		}

		if finalPayload == nil {
			panic("cannot fit chunk into 1023KB image")
		}

		filename := fmt.Sprintf(
			"%s_%05d_%d.webp",
			prefix, chunkIndex, len(finalPayload),
		)

		die(os.WriteFile(filename, finalImage, 0644))

		fmt.Println("Written:", filename, "(", len(finalImage), "bytes )")

		offset += len(finalPayload)
		chunkIndex++
	}

	die(os.WriteFile(
		prefix+"_total.txt",
		[]byte(strconv.Itoa(chunkIndex)),
		0644,
	))

	fmt.Println("✅ Encoding complete")
}

/////////////////////////////
// DECODE
/////////////////////////////

type chunk struct {
	index int
	size  int
	file  string
}

func decode(prefix, output string) {

	totalBytes, err := os.ReadFile(prefix + "_total.txt")
	die(err)
	total, _ := strconv.Atoi(strings.TrimSpace(string(totalBytes)))

	files, err := filepath.Glob(prefix + "_*.webp")
	die(err)

	re := regexp.MustCompile(prefix + `_(\d+)_(\d+)\.webp`)

	var chunks []chunk
	for _, f := range files {
		m := re.FindStringSubmatch(f)
		if len(m) == 3 {
			i, _ := strconv.Atoi(m[1])
			s, _ := strconv.Atoi(m[2])
			chunks = append(chunks, chunk{i, s, f})
		}
	}

	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].index < chunks[j].index
	})

	if len(chunks) != total {
		panic("chunk count mismatch")
	}

	var compressed []byte

	for _, c := range chunks {

		f, err := os.Open(c.file)
		die(err)
		img, err := webp.Decode(f)
		die(err)
		f.Close()

		b := img.Bounds()
		raw := make([]byte, 0, b.Dx()*b.Dy()*3)

		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				raw = append(raw, byte(r>>8), byte(g>>8), byte(b>>8))
			}
		}

		compressed = append(compressed, raw[:c.size]...)
		fmt.Println("Read:", c.file)
	}

	fmt.Println("Decompressing...")
	out := zlibDecompress(compressed)
	die(os.WriteFile(output, out, 0644))

	fmt.Println("✅ Decoding complete")
}

/////////////////////////////
// MAIN
/////////////////////////////

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  encode <inputFile> <prefix>")
		fmt.Println("  decode <prefix> <outputFile>")
		return
	}

	switch os.Args[1] {
	case "encode":
		if len(os.Args) != 4 {
			fmt.Println("encode <inputFile> <prefix>")
			return
		}
		encode(os.Args[2], os.Args[3])

	case "decode":
		if len(os.Args) != 4 {
			fmt.Println("decode <prefix> <outputFile>")
			return
		}
		decode(os.Args[2], os.Args[3])

	default:
		fmt.Println("Unknown command")
	}
}
