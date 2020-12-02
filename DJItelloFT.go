package main

import (
	"fmt" // Formatted I/O
	"io" //  It provides basic interfaces to I/O primitives
	"os/exec" // To run the external commands.
	"strconv" // Package strconv implements conversions to and from string
	"time" //For time related operation
	"log"
    "image"
	"gobot.io/x/gobot" // Gobot Framework.
	"gobot.io/x/gobot/platforms/dji/tello" // DJI Tello package.
	"gocv.io/x/gocv" // GoCV package to access the OpenCV library.
	"golang.org/x/image/colornames"
)

//first variable enables tracking as long as the drone is not landing
var inFlight = false

//these variables used for tracking
var minBufX = 27
var minBufY = 39
var maxBufX = 93
var maxBufY = 51

/*
//would have been used for depth adjustment
var minFrame = 30
var maxFrame = 40
*/

// Frame size constant.
const (
	frameX    = 720
	frameY    = 960
	frameSize = frameX * frameY * 3
)

func main() {
	// Driver: Tello Driver
	drone := tello.NewDriver("8890")

	// OpenCV window to watch the live video stream from Tello.
	window := gocv.NewWindow("Tello")

	//classifier stuff
	classifier := gocv.NewCascadeClassifier()
	classifier.Load("haarcascade_frontalface_default.xml")
	defer classifier.Close()

	//FFMPEG command to convert the raw video from the drone.
	ffmpeg := exec.Command("ffmpeg", "-hwaccel", "auto", "-hwaccel_device", "opencl", "-i", "pipe:0",
		"-pix_fmt", "bgr24", "-s", strconv.Itoa(frameY)+"x"+strconv.Itoa(frameX), "-f", "rawvideo", "pipe:1")
	ffmpegIn, _ := ffmpeg.StdinPipe()
	ffmpegOut, _ := ffmpeg.StdoutPipe()



	work := func() {
		//Starting FFMPEG.
		fmt.Println("starting ffmpeg")
		if err := ffmpeg.Start(); err != nil {
			fmt.Println("there is an error")
			fmt.Println(err)
			return
		}

        // need this
        go func() {

        }()

		// Event: Listening the Tello connect event to start the video streaming.
		go drone.On(tello.ConnectedEvent, func(data interface{}) {
			fmt.Println("Connected to Tello.")
			drone.StartVideo()
			drone.SetVideoEncoderRate(tello.VideoBitRateAuto)
			drone.SetExposure(0)

			//For continued streaming of video.
			gobot.Every(500*time.Millisecond, func() {
				drone.StartVideo()

			})
		})

		//Event: Piping the video data into the FFMPEG function.
		go drone.On(tello.VideoFrameEvent, func(data interface{}) {
			//fmt.Println("receiving data")
			pkt := data.([]byte)
			if _, err := ffmpegIn.Write(pkt); err != nil {
				fmt.Println(err)
			}
		})

		//Have the Drone take off
		gobot.After(5*time.Second, func() {
			go drone.TakeOff()
			fmt.Println("Tello Taking Off...")
			inFlight = true
		})

		//Land the Drone after a given amount of time. Stop any tracking
		gobot.After(60*time.Second, func() {
			go drone.Land()
			fmt.Println("Tello Landing...")
			inFlight = false
		})

	}

	//Robot: Tello Drone
	robot := gobot.NewRobot("tello",
		[]gobot.Connection{},
		[]gobot.Device{drone},
		work,
	)

	// calling Start(false) lets the Start routine return immediately without an additional blocking goroutine
	robot.Start(false)

	// now handle video frames from ffmpeg stream in main thread, to be macOs friendly
	for {
		buf := make([]byte, frameSize)
//		fmt.Println("handle vid frames")
		if _, err := io.ReadFull(ffmpegOut, buf); err != nil {
//			fmt.Println("error, jumping up")
			fmt.Println(err)
			continue
		}

		img, _ := gocv.NewMatFromBytes(frameX, frameY, gocv.MatTypeCV8UC3, buf)
		if img.Empty() {
		    fmt.Println("Empty image")
			continue
		}

		faceDetect := gocv.NewMat()
		gocv.Resize( img, &faceDetect, image.Pt( 90, 120 ), 0, 0, gocv.InterpolationNearestNeighbor)

		//detect a face
		imageRectangles := classifier.DetectMultiScale( faceDetect )
		//fmt.Println(imageRectangles)

		for _, rect := range imageRectangles {
			log.Println("found a face,", rect)
			gocv.Rectangle(&faceDetect, rect, colornames.Cadetblue, 3)

            //next line could be used for depth
			//fmt.Println("X is", rect.Size().X)

            //used to get the dimensions of the rectangle
			minFaceX := rect.Min.X
			maxFaceX := rect.Max.X
			minFaceY := rect.Min.Y
			maxFaceY := rect.Max.Y
/*
            //this was used to try to correct for depth. Did not work
			sideX := maxFaceX - minFaceX
			sideY := maxFaceY - minFaceY
			sizeF := ((sideX^2) + (sideY^2))^(1/2)
*/

			//use values to move drone to center within buffer
            //only run if this boolean is true
            if inFlight{
    			if minFaceX < minBufX{
    			    drone.CounterClockwise(15)
                } else if maxFaceX > maxBufX {
                    drone.Clockwise(15)
                } else {
                    drone.Clockwise(0)
                }

    			if minFaceY < minBufY{
    			    drone.Up(15)
                } else if maxFaceY > maxBufY {
                    drone.Down(15)
                } else {
                    drone.Up(0)
                }
/*
                //was also used for distance. Did not work correctly
                if sizeF < minFrame{
                    drone.Forward(15)
                } else if sizeF > maxFrame{
                    drone.Backward(15)
                } else{
                    drone.Forward(0)
                }
*/
            }

		}

		window.IMShow(faceDetect)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}

