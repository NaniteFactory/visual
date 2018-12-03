package visual // namespace

import (
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"
	"time"
	"unsafe"

	glfw "github.com/go-gl/glfw/v3.2/glfw"
	kuji "github.com/nanitefactory/amidakuji/glossary"
	"github.com/nanitefactory/visual/actors"
	"github.com/nanitefactory/visual/jukebox"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"github.com/sqweek/dialog"
	"golang.org/x/image/colornames"
)

// -------------------------------------------------------------------------
// Itf export (Actor)

// Actor is what a Visualizer visualizes.
// Actor updates and draws itself. It acts as a game (virtual) object.
//
// The main-thread, called Visualizer,
// will do what's shown below, every single frame.
//
// func (v *Visualizer) _NextFrame(dt float64) {
// 	// ---------------------------------------------------
// 	// 1. update - calc state of game (virtual) objects each frame
// 	v._Update(dt)
// 	v.fpsw.Poll()
//
// 	// ---------------------------------------------------
// 	// 2. draw on window
// 	v.window.Clear(v.bg) // clear canvas
// 	v._Draw()            // then draw
//
// 	// ---------------------------------------------------
// 	// 3. update window - always end with it
// 	v.window.Update()
// 	<-v.vsync
// }
//
type Actor interface {
	Drawer
	Updater
}

// Drawer draws itself on a target canvas.
//
// The main-thread will do what's shown below every single frame.
//
// // Canvas a game (virtual) world
// t.SetMatrix(v.camera.Transform())
//
// // For all actors, Draw() in an order.
// for i := range v.actors {
// 	v.actors[i].Draw(t)
// }
//
type Drawer interface {
	// Draw obligatorily invoked by Visualizer on main thread.
	Draw(t pixel.Target)
}

// Updater updates itself with the delta time given, every frame on main-thread.
//
// The main-thread will do what's shown below every single frame.
//
// // For all actors, Update() in an order.
// for i := range v.actors {
// 	v.actors[i].Update(dt)
// }
//
type Updater interface {
	// Update obligatorily invoked by Visualizer on main thread.
	Update(dt float64)
}

// -------------------------------------------------------------------------
// Config

// Config is just an argument of NewVisualizer() that defines a new visualizer.
type Config struct {
	Bg                  pixel.RGBA
	OnDrawn             func(t pixel.Target)
	OnUpdated           func(dt float64)
	OnResized           func(width float64, height float64)
	OnPaused            func()
	OnResumed           func()
	OnClose             func()
	OnHandlingEvents    func(dt float64, window *pixelgl.Window)
	OnLogging           func(args ...interface{})
	WinCentered         bool
	Undecorated         bool
	Title               string
	Version             string
	Width               float64
	Height              float64
	WinWidth            float64
	WinHeight           float64
	InitialZoomLevel    float64
	InitialRotateDegree float64
}

// PosCenterGame defines the world center in game position.
func (c Config) PosCenterGame() pixel.Vec {
	return pixel.V(c.Width/2, c.Height/2)
}

// -------------------------------------------------------------------------
// Visualizer

// Visualizer is a main-thread that visualizes stuff.
//
// Visualizer manages:
// 1) a window,
// 2) Actors,
// 3) and a game (visualizer) system along with vsync/fps/dt/camera.
//
// Controls provided by default: ESC, Tab, Enter, Space, Arrows, Mouse input L/R/Wheel
//
type Visualizer struct { // aka. game
	// something system, somthing runtime
	window *pixelgl.Window // lazy init
	bg     pixel.RGBA
	camera *kuji.Camera // lazy init
	fpsw   *actors.FPSWatch
	dtw    kuji.DtWatch
	vsync  <-chan time.Time // lazy init
	// game (visualizer) state
	isScalpelMode bool
	// drawings
	mutex      sync.Mutex // actors must be locked up
	actors     []Actor
	huds       []Actor
	explosions *actors.Explosions
	// callbacks
	onDrawn          func(t pixel.Target)
	onUpdated        func(dt float64)
	onResized        func(width float64, height float64)
	onPaused         func()
	onResumed        func()
	onClose          func()
	onHandlingEvents func(dt float64, window *pixelgl.Window)
	onLogging        func(args ...interface{})
	// other initial user settings
	winCentered         bool
	undecorated         bool
	title               string
	version             string
	width               float64
	height              float64
	winWidth            float64 // The screen width, not the game width.
	winHeight           float64
	initialZoomLevel    float64
	initialRotateDegree float64
}

// NewVisualizer is a constructor.
func NewVisualizer(cfg Config, _huds []Actor, _actors ...Actor) *Visualizer {
	v := Visualizer{
		bg:                  cfg.Bg,
		fpsw:                actors.NewFPSWatchSimple(pixel.V(cfg.WinWidth, cfg.WinHeight), kuji.Top, kuji.Right),
		actors:              make([]Actor, len(_actors)+1), // +1 for the default actor; explosions
		huds:                make([]Actor, len(_huds)+1),   // +1 for the default actor(hud); fpswatch
		explosions:          actors.NewExplosions(cfg.Width, cfg.Height, nil, 4),
		onPaused:            cfg.OnPaused,
		onResumed:           cfg.OnResumed,
		onDrawn:             cfg.OnDrawn,
		onUpdated:           cfg.OnUpdated,
		onResized:           cfg.OnResized,
		onClose:             cfg.OnClose,
		onHandlingEvents:    cfg.OnHandlingEvents,
		onLogging:           cfg.OnLogging,
		winCentered:         cfg.WinCentered,
		undecorated:         cfg.Undecorated,
		title:               cfg.Title,
		version:             cfg.Version,
		width:               cfg.Width,
		height:              cfg.Height,
		winWidth:            cfg.WinWidth,
		winHeight:           cfg.WinHeight,
		initialZoomLevel:    cfg.InitialZoomLevel,
		initialRotateDegree: cfg.InitialRotateDegree,
	}

	// Actors in screen coords.
	for i := range _huds {
		v.huds[i] = _huds[i]
	}

	// Add fpsw to the last element of huds(actors).
	v.huds[len(v.huds)-1] = v.fpsw

	// Actors in game coords.
	for i := range _actors {
		v.actors[i] = _actors[i]
	}

	// Add explosions to the last element of actors.
	v.actors[len(v.actors)-1] = v.explosions

	// This will be finalized (cleaned-up) when the window gets closed.
	err := jukebox.Initialize()
	if err != nil {
		v.logPrintln("This function successfully returns, though there was an error: ", err)
	}

	return &v
}

// Draw () instructs this visualizer to draw its Actors.
func (v *Visualizer) _Draw() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// The target canvas Draw() draws on is called t.
	var t pixel.BasicTarget
	t = v.window

	// ---------------------------------------------------
	// 1. canvas a game world
	t.SetMatrix(v.camera.Transform())

	// For all actors, Draw() in order.
	for i := range v.actors {
		v.actors[i].Draw(t)
	}

	// Custom action after that all actors got drawn.
	if v.onDrawn != nil {
		v.onDrawn(t)
	}

	// ---------------------------------------------------
	// 2. canvas a screen
	t.SetMatrix(pixel.IM)

	// Draw()s in an order.
	for i := range v.huds {
		v.huds[i].Draw(t)
	}
}

// Update () instructs this visualizer to update its Actors.
func (v *Visualizer) _Update(dt float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// The camera would and should update every frame.
	v.camera.Update(dt)

	// For all actors, Update() in order.
	for i := range v.actors {
		v.actors[i].Update(dt)
	}

	// For all huds(actors), Update() in order.
	for i := range v.huds {
		v.huds[i].Update(dt)
	}

	// Custom action after that all actors got updated.
	if v.onUpdated != nil {
		v.onUpdated(dt)
	}
}

func (v *Visualizer) _OnResize(width, height float64) {
	v.camera.SetScreenBound(pixel.R(0, 0, width, height))
	v.fpsw.SetPos(pixel.V(width, height), kuji.Top, kuji.Right)
	if v.onResized != nil {
		v.onResized(width, height)
	}
}

// Pause everything going on.
func (v *Visualizer) Pause() {
	if v.onPaused != nil {
		v.onPaused()
	}
}

// Resume after pause.
func (v *Visualizer) Resume() {
	v.dtw.Dt()

	if v.onResumed != nil {
		v.onResumed()
	}
}

func (v *Visualizer) _SetFullScreenMode(on bool) {
	if on {
		monitor := pixelgl.PrimaryMonitor()
		width, height := monitor.Size()
		v.window.SetMonitor(monitor)
		go func(width, height float64) {
			v._OnResize(width, height)
		}(width, height)
	} else if !on { // off
		v.window.SetMonitor(nil)
	} else {
		panic(errors.New("it may be thread"))
	}
}

func (v *Visualizer) logPrintln(args ...interface{}) {
	if v.onLogging != nil {
		v.onLogging(args...)
	}
	log.Println(args...)
}

// -------------------------------------------------------------------------
// Read only methods

// WindowDeep is a hacky way to access `glfw.Window`.
// It returns (window *glfw.Window) which is an unexported member inside a (*pixelgl.Window).
func (v *Visualizer) _WindowDeep() (baseWindow *glfw.Window) {
	return *(**glfw.Window)(unsafe.Pointer(reflect.Indirect(reflect.ValueOf(v.window)).FieldByName("window").UnsafeAddr()))
}

// -------------------------------------------------------------------------
// Run on main thread

// Run the game window and its event loop on main thread.
func (v *Visualizer) Run() {
	pixelgl.Run(func() {
		v._RunLazyInit()
		v._RunEventLoop()
	})
}

func (v *Visualizer) _RunLazyInit() {
	// This window will show up as soon as it is created.
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       v.title + "  (" + v.version + ")",
		Icon:        nil,
		Bounds:      pixel.R(0, 0, v.winWidth, v.winHeight),
		Monitor:     nil,
		Resizable:   true,
		Undecorated: v.undecorated,
		VSync:       false,
	})
	if err != nil {
		panic(err)
	}
	win.SetSmooth(true)

	if v.winCentered {
		MoveWindowToCenterOfPrimaryMonitor := func(win *pixelgl.Window) {
			vmodes := pixelgl.PrimaryMonitor().VideoModes()
			vmodesLast := vmodes[len(vmodes)-1]
			biggestResolution := pixel.R(0, 0, float64(vmodesLast.Width), float64(vmodesLast.Height))
			win.SetPos(biggestResolution.Center().Sub(win.Bounds().Center()))
		}
		MoveWindowToCenterOfPrimaryMonitor(win)
	}

	// lazy init vars
	v.window = win
	v.camera = kuji.NewCamera(pixel.V(v.width/2, v.height/2), v.window.Bounds())

	// register callback
	windowGL := v._WindowDeep()
	windowGL.SetSizeCallback(func(_ *glfw.Window, width int, height int) {
		v._OnResize(float64(width), float64(height))
	})
	windowGL.SetCloseCallback(func(w *glfw.Window) {
		err := jukebox.Finalize()
		if err != nil {
			v.logPrintln(err)
		}
		if v.onClose != nil {
			v.onClose()
		}
	})

	// time manager
	v.vsync = time.Tick(time.Second / 120)
	v.fpsw.Start()
	v.dtw.Start()

	// so-called loading
	{
		v.window.Clear(colornames.Brown)
		screenCenter := v.window.Bounds().Center()
		txt := text.New(screenCenter, kuji.NewAtlas("", 36, nil))
		txt.WriteString("Loading...")
		txt.Draw(v.window, pixel.IM)
		v.window.Update()
	}
	v._NextFrame(v.dtw.Dt()) // Give it a blood pressure.
	v._NextFrame(v.dtw.Dt()) // Now the oxygenated blood will start to pump through its vein.
	// Do whatever you want after that...

	// from user setting
	v.camera.Zoom(float64(v.initialZoomLevel))
	v.camera.Rotate(v.initialRotateDegree)
}

func (v *Visualizer) _RunEventLoop() {
	for v.window.Closed() != true { // Your average event loop in main thread.
		// Notice that all function calls as go routine are non-blocking, but the others will block the main thread.

		// ---------------------------------------------------
		// 0. dt
		dt := v.dtw.Dt()

		// ---------------------------------------------------
		// 1. handling events
		v._HandlingEvents(dt)

		// ---------------------------------------------------
		// 2. move on
		v._NextFrame(dt)

	} // for
} // func

func (v *Visualizer) _HandlingEvents(dt float64) {
	// Notice that all function calls as go routine are non-blocking, but the others will block the main thread.

	// custom event handler
	if v.onHandlingEvents != nil {
		v.onHandlingEvents(dt, v.window)
	}

	// system
	if v.window.JustReleased(pixelgl.KeyEscape) {
		v.window.SetClosed(true)
	}
	if v.window.JustReleased(pixelgl.KeySpace) {
		v.Pause()
		dialog.Message("%s", "Pause").Title("PPAP").Info()
		v.Resume()
	}
	if v.window.JustReleased(pixelgl.KeyTab) {
		if v.window.Monitor() == nil {
			v._SetFullScreenMode(true)
		} else {
			v._SetFullScreenMode(false)
		}
	}

	// scalpel mode
	if v.window.JustReleased(pixelgl.MouseButtonRight) {
		go func() {
			v.isScalpelMode = !v.isScalpelMode
		}()
	}
	if v.window.JustReleased(pixelgl.MouseButtonLeft) {
		// ---------------------------------------------------
		if !jukebox.IsPlaying() {
			jukebox.Play()
		}

		// ---------------------------------------------------
		posWin := v.window.MousePosition()
		posGame := v.camera.Unproject(posWin)
		go func() {
			v.explosions.ExplodeAt(pixel.V(posGame.X, posGame.Y), pixel.V(10, 10))
		}()

		// ---------------------------------------------------
		if v.isScalpelMode {
			// strTitle := fmt.Sprint(posGame.X, ", ", posGame.Y) //
			strDlg := fmt.Sprint(
				"camera angle in degree: ", (v.camera.Angle()/math.Pi)*180, "\r\n", "\r\n",
				"camera coordinates: ", v.camera.XY().X, v.camera.XY().Y, "\r\n", "\r\n",
				"game clock: ", v.dtw.GetTimeStarted(), "\r\n", "\r\n",
				"mouse click coords in screen pos: ", posWin.X, posWin.Y, "\r\n", "\r\n",
				"mouse click coords in game pos: ", posGame.X, posGame.Y,
			)
			go func() {
				// v.window.SetTitle(strTitle) //
				dialog.Message("%s", strDlg).Title("MouseButtonLeft").Info()
			}()
		}
	}

	// camera
	if v.window.JustReleased(pixelgl.KeyEnter) {
		go func() {
			v.camera.Rotate(-90)
		}()
	}
	if v.window.Pressed(pixelgl.KeyRight) {
		go func(dt float64) { // This camera will go diagonal while the case is in middle of rotating the camera.
			v.camera.Move(pixel.V(1000*dt, 0).Rotated(-v.camera.Angle()))
		}(dt)
	}
	if v.window.Pressed(pixelgl.KeyLeft) {
		go func(dt float64) {
			v.camera.Move(pixel.V(-1000*dt, 0).Rotated(-v.camera.Angle()))
		}(dt)
	}
	if v.window.Pressed(pixelgl.KeyUp) {
		go func(dt float64) {
			v.camera.Move(pixel.V(0, 1000*dt).Rotated(-v.camera.Angle()))
		}(dt)
	}
	if v.window.Pressed(pixelgl.KeyDown) {
		go func(dt float64) {
			v.camera.Move(pixel.V(0, -1000*dt).Rotated(-v.camera.Angle()))
		}(dt)
	}
	{ // if scrolled
		zoomLevel := v.window.MouseScroll().Y
		go func() {
			v.camera.Zoom(zoomLevel)
		}()
	}
}

func (v *Visualizer) _NextFrame(dt float64) {
	// ---------------------------------------------------
	// 1. update - calc state of game objects each frame
	v._Update(dt)
	v.fpsw.Poll()

	// ---------------------------------------------------
	// 2. draw on window
	v.window.Clear(v.bg) // clear canvas
	v._Draw()            // then draw

	// ---------------------------------------------------
	// 3. update window - always end with it
	v.window.Update()
	<-v.vsync
}
