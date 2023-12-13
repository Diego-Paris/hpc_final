package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type PerformanceData struct {
	ImageNumber    int
	SequentialTime time.Duration
	ParallelTime   time.Duration
}

// PrintExecutionTimesTable prints a table of execution times
func PrintExecutionTimesTable(performanceData []PerformanceData) {
	fmt.Println("Image\tSequential Time (s)\tParallel Time (s)")
	fmt.Println("--------------------------------------------------")

	for _, data := range performanceData {
		fmt.Printf("%d\t%.6f\t\t%.6f\n", data.ImageNumber, data.SequentialTime.Seconds(), data.ParallelTime.Seconds())
	}
}

// Convert to Black and White
func toBlackAndWhite(img image.Image) *image.Gray {
	bounds := img.Bounds()
	grayScale := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, _ := originalColor.RGBA()
			grayValue := uint8((r + g + b) / 3 >> 8) // Average of RGB
			grayScale.Set(x, y, color.Gray{Y: grayValue})
		}
	}
	return grayScale
}

// Get neighborhood pixel values
func getNeighborhood(img *image.Gray, x, y, size int) []uint8 {
	var values []uint8
	for dy := -size; dy <= size; dy++ {
		for dx := -size; dx <= size; dx++ {
			nx, ny := x+dx, y+dy
			if nx >= 0 && ny >= 0 && nx < img.Rect.Max.X && ny < img.Rect.Max.Y {
				values = append(values, img.GrayAt(nx, ny).Y)
			}
		}
	}
	return values
}

// Median Filter (Sequential)
func medianFilterSequential(img *image.Gray) *image.Gray {
	bounds := img.Bounds()
	output := image.NewGray(bounds)
	filterSize := 1 // You can adjust this size

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			neighborhood := getNeighborhood(img, x, y, filterSize)
			sort.Slice(neighborhood, func(i, j int) bool { return neighborhood[i] < neighborhood[j] })
			median := neighborhood[len(neighborhood)/2]
			output.SetGray(x, y, color.Gray{Y: median})
		}
	}
	return output
}

// Median Filter (Parallel)
func medianFilterParallel(img *image.Gray, chunkSize int) *image.Gray {
	bounds := img.Bounds()
	output := image.NewGray(bounds)
	filterSize := 1 // You can adjust this size
	var wg sync.WaitGroup

	for y := bounds.Min.Y; y < bounds.Max.Y; y += chunkSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += chunkSize {
			wg.Add(1)
			go func(x, y int) {
				defer wg.Done()
				for cy := y; cy < y+chunkSize && cy < bounds.Max.Y; cy++ {
					for cx := x; cx < x+chunkSize && cx < bounds.Max.X; cx++ {
						neighborhood := getNeighborhood(img, cx, cy, filterSize)
						sort.Slice(neighborhood, func(i, j int) bool { return neighborhood[i] < neighborhood[j] })
						median := neighborhood[len(neighborhood)/2]
						output.SetGray(cx, cy, color.Gray{Y: median})
					}
				}
			}(x, y)
		}
	}
	wg.Wait()

	return output
}

// Measure the execution time
func measureTime(function func() *image.Gray) time.Duration {
	start := time.Now()
	function()
	return time.Since(start)
}

func saveImage(img image.Image, folder, filename string) {
	// Check if the directory exists, if not create it
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		os.Mkdir(folder, os.ModePerm)
	}

	// Save the image
	outFile, err := os.Create(filepath.Join(folder, filename))
	if err != nil {
		log.Fatalf("failed to create file: %v", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, img); err != nil {
		log.Fatalf("failed to encode image: %v", err)
	}
}

func main() {
	fmt.Println("Running Median Filter, please wait...")
	p := plot.New()
	p.Title.Text = "Performance Comparison"
	p.X.Label.Text = "Image Number"
	p.Y.Label.Text = "Time (s)"
	var performanceData []PerformanceData

	sequentialPoints := make(plotter.XYs, 24)
	parallelPoints := make(plotter.XYs, 24)

	for i := 1; i <= 24; i++ {
		filename := fmt.Sprintf("kodim%02d.png", i)
		inFile, err := os.Open(filepath.Join("dataset", filename))
		if err != nil {
			log.Fatalf("failed to open %s: %v", filename, err)
		}

		img, _, err := image.Decode(inFile)
		inFile.Close()
		if err != nil {
			log.Fatalf("failed to decode %s: %v", filename, err)
		}

		bwImage := toBlackAndWhite(img)

		// Save black and white image with noise
		saveImage(bwImage, "dataset-w-noise", filename)

		// Measure sequential processing time
		seqTime := measureTime(func() *image.Gray {
			return medianFilterSequential(bwImage)
		})

		sequentialOutput := medianFilterSequential(bwImage)
		saveImage(sequentialOutput, "dataset-output", fmt.Sprintf("sequential-%s", filename))

		// Measure parallel processing time
		parallelTime := measureTime(func() *image.Gray {
			return medianFilterParallel(bwImage, 45) // Adjust the chunkSize value as needed
		})
		parallelOutput := medianFilterParallel(bwImage, 45) // Adjust the chunkSize
		saveImage(parallelOutput, "dataset-output", fmt.Sprintf("parallel-%s", filename))

		data := PerformanceData{
			ImageNumber:    i,
			SequentialTime: seqTime,
			ParallelTime:   parallelTime,
		}
		performanceData = append(performanceData, data)

		//fmt.Printf("Image %d - Sequential Time: %v seconds\n", i, seqTime.Seconds())
		//fmt.Printf("Image %d - Parallel Time: %v seconds\n", i, parallelTime.Seconds())
		sequentialPoints[i-1] = plotter.XY{X: float64(i), Y: seqTime.Seconds()}
		parallelPoints[i-1] = plotter.XY{X: float64(i), Y: parallelTime.Seconds()}
	}

	seqLine, seqPoints, err := plotter.NewLinePoints(sequentialPoints)
	if err != nil {
		log.Fatalf("failed to create line points for sequential: %v", err)
	}
	seqLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red line for sequential

	parLine, parPoints, err := plotter.NewLinePoints(parallelPoints)
	if err != nil {
		log.Fatalf("failed to create line points for parallel: %v", err)
	}
	parLine.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255} // Blue line for parallel

	// Adjust the legend position
	p.Legend.Top = false
	p.Legend.Left = false
	p.Legend.XOffs = vg.Points(-500) // You can adjust this for fine positioning
	p.Legend.YOffs = vg.Points(-30)  // You can adjust this for fine positioning

	// Add the lines and points to the plot
	p.Add(seqLine, seqPoints)
	p.Add(parLine, parPoints)

	// Add legend entries
	p.Legend.Add("Sequential", seqLine, seqPoints)
	p.Legend.Add("Parallel", parLine, parPoints)

	// Save the plot
	if err := p.Save(8*vg.Inch, 4*vg.Inch, "performance_comparison.png"); err != nil {
		log.Fatalf("failed to save plot: %v", err)
	}

	PrintExecutionTimesTable(performanceData)
}
