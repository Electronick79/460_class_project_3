package main

import (
	"fmt" // Formatted I/O
	"io" //  It provides basic interfaces to I/O primitives
	"os/exec" // To run the external commands.
	"strconv" // Package strconv implements conversions to and from string
	"time" //For time related operation
	//"log"
	"image"
	"image/color"

	"gobot.io/x/gobot" // Gobot Framework.
	"gobot.io/x/gobot/platforms/dji/tello" // DJI Tello package.
	"gocv.io/x/gocv" // GoCV package to access the OpenCV library.
//	"golang.org/x/image/colornames"
)

// Frame size constant.
const (
	frameX    = 960
	frameY    = 720
	frameSize = frameX * frameY * 3
)

func main() {
	// Driver: Tello Driver
	drone := tello.NewDriver("8890")

    //window := opencv.NewWindowDriver()
	// OpenCV window to watch the live video stream from Tello.
	window := gocv.NewWindow("Tello")


	//classifier stuff
	classifier := gocv.NewCascadeClassifier()
	//classifier.Load("haarcascade_frontalface_default.xml")
	classifier.Load("eyelook.xml")
	defer classifier.Close()

	blue := color.RGBA{0, 0, 255, 0}


	//FFMPEG command to convert the raw video from the drone.
	ffmpeg := exec.Command("ffmpeg", "-hwaccel", "auto", "-hwaccel_device", "opencl", "-i", "pipe:0",
		"-pix_fmt", "bgr24", "-s", strconv.Itoa(frameX)+"x"+strconv.Itoa(frameY), "-f", "rawvideo", "pipe:1")
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
			gobot.Every(250*time.Millisecond, func() {
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

		//TakeOff the Drone.
		go gobot.After(5*time.Second, func() {
			drone.TakeOff()
			fmt.Println("Tello Taking Off...")
		})

		//Land the Drone.
		go gobot.After(15*time.Second, func() {
			drone.Land()
			fmt.Println("Tello Landing...")
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

	// now handle video frames from ffmpeg stream in main thread, runs continuously
	go for {
		buf := make([]byte, frameSize)
//		fmt.Println("handle vid frames")
		if _, err := io.ReadFull(ffmpegOut, buf); err != nil {
//			fmt.Println("error, jumping up")
			//EOF is error being printed, fixed as of 11/16
			fmt.Println(err)
			continue
		}

		img, _ := gocv.NewMatFromBytes(frameY, frameX, gocv.MatTypeCV8UC3, buf)
		if img.Empty() {
		    fmt.Println("Empty image")
			continue
		}

/*
        //code from Teams example
		//detect a face
		imageRectangles := classifier.DetectMultiScale(img)

		for _, rect := range imageRectangles {
			log.Println("found a face,", rect)
			gocv.Rectangle(&img, rect, colornames.Cadetblue, 3)
		}
*/
/*
        //code from the internet
		rects := classifier.DetectMultiScale(img)
		fmt.Printf("found %d faces\n", len(rects))

        //this loop causes the video feed to lag behind the images being sent back.
        //when run, looks like only a still image.
		for _, r := range rects {
			gocv.Rectangle(&img, r, blue, 3)

			size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
			return
		}
*/
		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}