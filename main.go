package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

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

func main() {
    p := plot.New()
    p.Title.Text = "Performance Comparison"
    p.X.Label.Text = "Image Number"
    p.Y.Label.Text = "Time (s)"

    sequentialPoints := make(plotter.XYs, 24)
    parallelPoints := make(plotter.XYs, 24)

    for i := 1; i <= 24; i++ {
        filename := fmt.Sprintf("dataset/kodim%02d.png", i)
        inFile, err := os.Open(filename)
        if err != nil {
            log.Fatalf("failed to open %s: %v", filename, err)
        }

        img, _, err := image.Decode(inFile)
        inFile.Close()
        if err != nil {
            log.Fatalf("failed to decode %s: %v", filename, err)
        }

        bwImage := toBlackAndWhite(img)

        // Measure sequential processing time
        seqTime := measureTime(func() *image.Gray {
            return medianFilterSequential(bwImage)
        })

        // Measure parallel processing time
        parallelTime := measureTime(func() *image.Gray {
            return medianFilterParallel(bwImage, 45) // Adjust the chunkSize value as needed
        })

        sequentialPoints[i-1] = plotter.XY{X: float64(i), Y: seqTime.Seconds()}
        parallelPoints[i-1] = plotter.XY{X: float64(i), Y: parallelTime.Seconds()}
    }

    seqLine, seqPoints, err := plotter.NewLinePoints(sequentialPoints)
    if err != nil {
        log.Fatalf("failed to create line points for sequential: %v", err)
    }
    seqLine.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red line for sequential
    p.Add(seqLine, seqPoints)

    parLine, parPoints, err := plotter.NewLinePoints(parallelPoints)
    if err != nil {
        log.Fatalf("failed to create line points for parallel: %v", err)
    }
    parLine.Color = color.RGBA{R: 0, G: 0, B: 255, A: 255} // Blue line for parallel
    p.Add(parLine, parPoints)

    p.Legend.Add("Sequential", seqLine, seqPoints)
    p.Legend.Add("Parallel", parLine, parPoints)

    if err := p.Save(8*vg.Inch, 4*vg.Inch, "performance_comparison.png"); err != nil {
        log.Fatalf("failed to save plot: %v", err)
    }
}
