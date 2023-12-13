# HPC Salt and Pepper Noise Filter

This project applies a median filter to a set of images, both sequentially and in parallel, to demonstrate performance differences. The images are first converted to black and white, and then processed. The results, along with the performance comparison, are saved as images.

---

## Pre-requisites
- Go (Golang) installed on your system. You can download and install Go from https://golang.org/dl/.
- Basic knowledge of running Go scripts.

## Installation
Clone the Repository: Start by cloning this repository to your local machine using:
```bash
git clone https://github.com/Diego-Paris/hpc_final.git
```

## Navigate to the project directory
```bash
cd path/to/project
```

## Setting up dependencies
This project requires the Gonum Plot package for creating plots. Install it by running:
```bash
go get -u gonum.org/v1/plot/...
```

## Running the script
To run the script, use the following command in the project root:
```bash
go run main.go
```
This will process the images, apply median filters, and save the outputs in the dataset-w-noise and dataset-output directories. It will also generate a performance comparison plot as performance_comparison.png.

## Output
- Black and white images with noise will be saved in dataset-w-noise.
- Images processed with median filters (both sequential and parallel) will be saved in dataset-output.
- A plot comparing the performance of sequential vs. parallel processing will be saved as performance_comparison.png.

## Troubleshooting
If you encounter any issues with running the script, make sure all dependencies are properly installed and that the dataset directory contains the correct images.