package main

import (
	"bytes"
	"codejester27/cmps401fa2024/web-app/processing"
	"encoding/base64"
	"encoding/json"

	"image"
	"image/color"
	"image/jpeg"
	"image/png"

	"log"
	"net/http"
	"net/url"
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

	newImg, regionCount, _ := processing.SimplifyImage(img)

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
					"regionCount": regionCount,
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

	newImg, _, regionMap := processing.SimplifyImage(img)

	numRegions := len(regionMap.GetRegions())
	data := make([]RegionData, 0, numRegions)

	for i := 0; i < numRegions; i++ {
		region := regionMap.GetRegion(processing.RegionIndex(i))

		minX, minY := processing.FindRegionPosition(region)
		regionColor := processing.GetColorOfRegion(region, newImg)

		regionImage := image.NewNRGBA(region.GetBounds())

		for j := 0; j < len(region); j++ {
			regionImage.Set(int(region[j].X), int(region[j].Y), regionColor)
		}

		buf := new(bytes.Buffer)
		if err := png.Encode(buf, regionImage); err != nil {
			panic(err)
		}
		base64Region := base64.StdEncoding.EncodeToString(buf.Bytes())

		mesh, err := region.CreateMesh()
		if err != nil {
			continue
		}
		optimizedMesh := processing.StraightOpt(mesh)
		r := RegionData{i, regionColor, minX, minY, base64Region, optimizedMesh}
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
	RegionNumber int                 `json:"regionNumber"`
	RegionColor  color.Color         `json:"regionColor"`
	CornerX      int                 `json:"cornerX"`
	CornerY      int                 `json:"cornerY"`
	RegionImage  string              `json:"regionImage"`
	Mesh         []processing.Vertex `json:"mesh"`
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
}

func connectWebsocket(c *gin.Context) {
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

func main() {
	log.Printf("Running with %d CPUs\n", runtime.NumCPU())

	router := gin.New()
	router.SetTrustedProxies(nil)
	router.Use(gin.Recovery())

	logged := router.Group("") // I don't like seeing auth tokens in my terminal so we're not logging the websocket requests
	logged.Use(gin.Logger())

	// cors
	logged.Use(func(ctx *gin.Context) {
		origin := ctx.Request.Header.Get("Origin")
		originUrl, err := url.Parse(origin)
		if err != nil {
			return
		}
		hostname := originUrl.Hostname()

		switch hostname {
		case "cmps401fa2024.onrender.com", "www.boardmesh.app", "boardmesh.app", "localhost":
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
		}
	})

	logged.StaticFile("/", "./website/homepage.html")
	logged.Static("/static", "./website")
	logged.Static("/boardwalk", "./exported-game")
	logged.StaticFile("/manual", "./exported-manual/Prototype User Manual.pdf")
	logged.POST("/api/simplify", simplifyImage)
	logged.POST("/api/build-level", buildLevel)
	router.GET("/api/ws", connectWebsocket)

	port := Port
	if port == "" {
		port = "8080"
	}
	router.Run(":" + port)
}
