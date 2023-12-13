package main

import (
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

// Main function
func main() {
    // Load the image
    inFile, err := os.Open("dataset/kodim21.png")
    if err != nil {
        log.Fatalf("failed to open: %v", err)
    }
    defer inFile.Close()
    img, _, err := image.Decode(inFile)
    if err != nil {
        log.Fatalf("failed to decode: %v", err)
    }

    // Convert to black and white
    bwImage := toBlackAndWhite(img)

    // Measure sequential processing time
    seqTime := measureTime(func() *image.Gray {
        return medianFilterSequential(bwImage)
    })

    // Measure parallel processing time
    parallelTime := measureTime(func() *image.Gray {
        return medianFilterParallel(bwImage, 10) // Adjust the chunkSize value as needed
    })

    // Plotting the results
    p := plot.New()
    // if err != nil {
    //     log.Fatalf("failed to create plot: %v", err)
    // }

    p.Title.Text = "Performance Comparison"
    p.X.Label.Text = "Method"
    p.Y.Label.Text = "Time (s)"

    // Prepare data for the bar chart
    values := make(plotter.Values, 2)
    values[0] = seqTime.Seconds()
    values[1] = parallelTime.Seconds()

    // Create bar chart
    bars, err := plotter.NewBarChart(values, vg.Points(20))
    if err != nil {
        log.Fatalf("failed to create bar chart: %v", err)
    }

    p.Add(bars)
    p.NominalX("Sequential", "Parallel")

    if err := p.Save(4*vg.Inch, 4*vg.Inch, "performance.png"); err != nil {
        log.Fatalf("failed to save plot: %v", err)
    }

    // Optionally, save the processed images
    // ...
}
