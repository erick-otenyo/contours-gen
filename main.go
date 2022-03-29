package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/jonas-p/go-shp"
	"github.com/lukeroth/gdal"
)

func getProjection(prj_filepath string) (wkt string, err error) {

	// open .prj file
	f, err := os.Open(prj_filepath)
	if err != nil {
		return wkt, err
	}

	// close the file after we are done
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// scan and return first line only immediately
		return sc.Text(), sc.Err()
	}

	return wkt, err
}

func main() {

	shpPath := "data/input/points/gwWells.shp"

	shape, err := shp.Open(shpPath)

	if err != nil {
		log.Fatal(err)
	}

	defer shape.Close()

	// fields from the attribute table (DBF)
	fields := shape.Fields()

	// projection from .prj file
	wktString, err := getProjection("data/input/points/gwWells.prj")
	spatialRef := gdal.CreateSpatialReference(wktString)

	if err != nil {
		log.Fatal(err)
	}

	var x, y, z []float64

	// loop through all points in the shapefile
	for shape.Next() {
		n, _ := shape.Shape()
		for k, f := range fields {
			val := shape.ReadAttribute(n, k)

			if f.String() == "Longitude" {
				fVal, err := strconv.ParseFloat(val, 32)
				if err != nil {
					log.Fatal(err)
				}
				x = append(x, fVal)
			}

			if f.String() == "Latitude" {
				fVal, err := strconv.ParseFloat(val, 32)
				if err != nil {
					log.Fatal(err)
				}
				y = append(y, fVal)
			}

			if f.String() == "SurfaceEle" {
				fVal, err := strconv.ParseFloat(val, 32)
				if err != nil {
					log.Fatal(err)
				}
				z = append(z, fVal)
			}
		}
	}

	// finding max and min values
	var xMin, xMax, yMin, yMax = math.MaxFloat64, -math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64
	for i := range x {
		if x[i] < xMin {
			xMin = x[i]
		}
		if x[i] > xMax {
			xMax = x[i]
		}
		if y[i] < yMin {
			yMin = y[i]
		}
		if y[i] > yMax {
			yMax = y[i]
		}
	}

	// desired raster resolution in degrees
	resolution := 2.6516228627319196e-05

	// get final image xSize and ySize using given resolution
	xSizeFloat := math.Abs((xMax - xMin) / resolution)
	ySizeFloat := math.Abs((yMax - yMin) / resolution)

	// round off to the nearest int
	xSize := int(xSizeFloat + 0.5)
	ySize := int(ySizeFloat + 0.5)

	// get pixel inc value
	pixelSize := math.Abs((xMax - xMin) / float64(xSize))

	var nX, nY uint = uint(xSize), uint(ySize)

	// create grid raster array
	data, err := gdal.GridCreate(
		gdal.GA_InverseDistancetoAPower,
		gdal.GridInverseDistanceToAPowerOptions{Power: 2},
		x, y, z,
		xMin, xMax, yMin, yMax,
		nX, nY,
		gdal.DummyProgress,
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	// Save grid raster values to geotiff file
	fmt.Println("Loading driver")
	driver, err := gdal.GetDriverByName("GTiff")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Creating dataset")
	dataset := driver.Create("test_10.tif", xSize, ySize, 1, gdal.Float64, nil)
	defer dataset.Close()

	fmt.Println("Converting to WKT")
	srString, err := spatialRef.ToWKT()

	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("Assigning projection: %s\n", srString)
	dataset.SetProjection(srString)

	fmt.Println("Setting geotransform")
	dataset.SetGeoTransform([6]float64{xMin, pixelSize, 0, yMax, 0, -pixelSize})

	fmt.Println("Getting raster band")
	raster := dataset.RasterBand(1)

	fmt.Println("Writing to raster band")
	raster.IO(gdal.Write, 0, 0, xSize, ySize, data, xSize, ySize, 0, 0)
}
