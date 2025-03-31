package main

import (
	"bytes"
	"codejester27/cmps401fa2024/web-app/processing"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"log"
	"net/http"
	"os"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const LISTENER_TOKEN_ENV = "LISTENER_TOKEN"
const PORT_ENV = "PORT"

var (
	ListenerToken = os.Getenv(LISTENER_TOKEN_ENV)
	Port          = os.Getenv(PORT_ENV)
)

var IMAGE_NO_SET_ERROR = errors.New("Cannot use an image that has no Set method in buildRegionMapForWebAPI")

type SettableImage = interface {
	image.Image
	Set(x, y int, color color.Color)
}

func buildRegionMapForWebAPI(img image.Image, options processing.RegionMapOptions) (regionMap *processing.RegionMap) {
	var removedColor color.Color
	if options.AllowWhite {
		removedColor = processing.Blank
	} else {
		removedColor = processing.White
	}

	regionMap = processing.BuildRegionMap(img, options, func(r processing.Region) bool {
		if len(r) >= processing.MINIMUM_NUMBER_OF_PIXELS_FOR_VALID_REGION {
			return true
		}
		if i, ok := img.(SettableImage); ok {
			for _, pixel := range r {
				i.Set(int(pixel.X), int(pixel.Y), removedColor)
			}
		} else {
			panic(IMAGE_NO_SET_ERROR)
		}
		return false
	})

	return regionMap
}

func simplifyImage(c *gin.Context) {
	c.SetAccepted("multipart/form-data")

	fileh, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": `File not found, be sure it is included under the "image" key of your form`})
		return
	}

	file, err := fileh.Open()
	if err != nil {
		panic(err)
	}

	contentType := fileh.Header.Get("Content-Type")
	var img image.Image
	switch contentType {
	case "image/png":
		img, err = png.Decode(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Could not decode your PNG image."})
			return
		}
	case "image/jpeg":
		img, err = jpeg.Decode(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Could not decode your JPEG image."})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Only JPEG and PNG images are accepted."})
		return
	}

	img, err = processing.ResizeImage(img)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": err.Error()})
		return
	}

	newImg := processing.SimplifyImage(img, processing.RegionMapOptions{NoColorSeparation: false, AllowWhite: false})

	regionMap := buildRegionMapForWebAPI(newImg, processing.RegionMapOptions{})

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, newImg); err != nil {
		panic(err)
	}
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())
	listenerHub.NotifyListeners(ListenerMessage{
		Type: "simplify",
		Attachments: []AttachedFile{
			{
				Name:          fileh.Filename,
				ContentType:   "image/png",
				Base64Content: base64Img,
				Meta: map[string]any{
					"regionCount": len(regionMap.GetRegions()),
				},
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
	c.Data(http.StatusOK, "image/png", buf.Bytes())
	c.Header("Content-Disposition", `attachment; filename="simplified-image.png"`)
}

func buildLevel(c *gin.Context) {
	c.SetAccepted("multipart/form-data")

	fileh, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": `File not found, be sure it is included under the "image" key of your form`})
		return
	}

	file, err := fileh.Open()
	if err != nil {
		panic(err)
	}

	preserveColor := c.Request.FormValue("preserveColor") == "true"
	noColorSeparation := c.Request.FormValue("noColorSeparation") == "true"
	allowWhite := c.Request.FormValue("allowWhite") == "true"

	contentType := fileh.Header.Get("Content-Type")
	var img image.Image
	switch contentType {
	case "image/png":
		img, err = png.Decode(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Could not decode your PNG image."})
			return
		}
	case "image/jpeg":
		img, err = jpeg.Decode(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Could not decode your JPEG image."})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "Only JPEG and PNG images are accepted."})
		return
	}

	img, err = processing.ResizeImage(img)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": err.Error()})
		return
	}

	opts := processing.RegionMapOptions{
		NoColorSeparation: noColorSeparation,
		AllowWhite:        allowWhite,
	}

	newImg := processing.SimplifyImage(img, opts)

	regionMap := buildRegionMapForWebAPI(newImg, opts)

	numRegions := len(regionMap.GetRegions())
	data := make([]RegionData, 0, numRegions)

	for i := 1; i < numRegions+1; i++ {
		region := regionMap.GetRegion(processing.RegionId(i))

		minX, minY := processing.FindRegionPosition(region)
		regionColor := processing.GetColorOfRegion(region, newImg, noColorSeparation)
		var regionColorString string

		switch regionColor {
		case processing.Red:
			regionColorString = "Red"
		case processing.Green:
			regionColorString = "Green"
		case processing.Blue:
			regionColorString = "Blue"
		case processing.Black:
			regionColorString = "Black"
		case processing.White:
			regionColorString = "White"
		}

		regionImage := image.NewNRGBA(region.GetBounds())

		if preserveColor {
			for j := 0; j < len(region); j++ {
				regionImage.Set(int(region[j].X), int(region[j].Y), img.At(int(region[j].X), int(region[j].Y)))
			}
		} else {
			for j := 0; j < len(region); j++ {
				regionImage.Set(int(region[j].X), int(region[j].Y), regionColor)
			}
		}

		buf := new(bytes.Buffer)
		if err := png.Encode(buf, regionImage); err != nil {
			panic(err)
		}
		base64Region := base64.StdEncoding.EncodeToString(buf.Bytes())

		shape, err := region.CreateShape()
		if err != nil {
			continue
		}
		optimizedShape := processing.StraightOpt(shape)
		r := RegionData{i, regionColor, regionColorString, minX, minY, base64Region, optimizedShape}
		data = append(data, r)
	}

	d, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	listenerHub.NotifyListeners(ListenerMessage{
		Type: "build-level",
		Attachments: []AttachedFile{
			{
				Name:          "level.json",
				ContentType:   "application/json",
				Base64Content: string(d),
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
	c.Data(http.StatusOK, "application/json", d)
}

type RegionData struct {
	RegionNumber      int                 `json:"regionNumber"`
	RegionColor       color.Color         `json:"regionColor"`
	RegionColorString string              `json:"regionColorString"`
	CornerX           int                 `json:"cornerX"`
	CornerY           int                 `json:"cornerY"`
	RegionImage       string              `json:"regionImage"`
	Shape             []processing.Vertex `json:"shape"`
}

type AttachedFile struct {
	Name          string         `json:"name"`
	ContentType   string         `json:"contentType"`
	Base64Content string         `json:"base64Content"`
	Meta          map[string]any `json:"meta,omitempty"`
}

type ListenerMessage struct {
	Type        string         `json:"type"`
	Content     string         `json:"content,omitempty"`
	Attachments []AttachedFile `json:"attachments,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type ListenerHub struct {
	listeners []chan ListenerMessage
	sync.Mutex
}

var listenerHub = ListenerHub{
	listeners: make([]chan ListenerMessage, 0, 5),
}

func (lh *ListenerHub) NotifyListeners(msg ListenerMessage) {
	lh.Lock()
	defer lh.Unlock()
	for _, l := range lh.listeners {
		l <- msg
	}
}

func (lh *ListenerHub) AddListener(listener chan ListenerMessage) {
	lh.Lock()
	defer lh.Unlock()
	lh.listeners = append(lh.listeners, listener)
}

func (lh *ListenerHub) RemoveListener(listener chan ListenerMessage) {
	lh.Lock()
	defer lh.Unlock()
	close(listener)
	lh.listeners = slices.DeleteFunc(lh.listeners, func(c chan ListenerMessage) bool {
		return c == listener
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func connectListenerWebsocket(c *gin.Context) {
	if c.Query("token") != ListenerToken {
		c.Status(401)
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}

	ch := make(chan ListenerMessage, 1)
	listenerHub.AddListener(ch)

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				listenerHub.RemoveListener(ch)
				conn.Close()
				break
			}
		}
	}()

	log.Printf("Listener connected %s", c.Request.RemoteAddr)
	defer log.Printf("Listener disconnected %s", c.Request.RemoteAddr)
	for m := range ch {
		err := conn.WriteJSON(m)
		if err != nil {
			log.Printf("Could not send %T: %s", m, err)
			listenerHub.RemoveListener(ch)
			conn.Close()
			break
		}
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, s := range needles {
		if strings.Contains(haystack, s) {
			return true
		}
	}
	return false
}

func main() {
	log.Printf("Running with %d CPUs\n", runtime.NumCPU())

	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())

	logged := router.Group("") // I don't like seeing auth tokens in my terminal so we're not logging the websocket requests
	logged.Use(gin.Logger())

	// cors
	logged.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
	})

	games := logged.Group("")

	games.Use(func(ctx *gin.Context) {
		ctx.Header("Cross-Origin-Embedder-Policy", "require-corp")
		ctx.Header("Cross-Origin-Opener-Policy", "same-origin")
	})

	logged.StaticFile("/", "./homepage/board-site/dist/index.html")
	logged.StaticFile("/board.png", "./homepage/board-site/dist/board.png")
	logged.Static("/assets", "./homepage/board-site/dist/assets")
	games.Static("/boardwalk", "./exported-game")
	logged.StaticFile("/manual", "./exported-manual/User Manual.pdf")
	games.Static("/boardbox", "./exported-boardbox")
	logged.POST("/api/simplify", simplifyImage)
	logged.POST("/api/build-level", buildLevel)
	router.GET("/api/ws", connectListenerWebsocket)

	router.NoRoute(gin.Logger(), func(ctx *gin.Context) {
		url := ctx.Request.URL.Path
		if containsAny(url, "wp", "wordpress", "admin", "php") {
			ctx.File("./misc-assets/teapot.txt")
			ctx.Status(http.StatusTeapot)
			return
		}
		ctx.File("./homepage/board-site/dist/index.html")
	})

	port := Port
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
