// Package visual provides a simple visualizer system that's based on faiface/pixel.
package visual

import (
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	glfw "github.com/go-gl/glfw/v3.2/glfw"
	"github.com/nanitefactory/visual/actors"
	"github.com/nanitefactory/visual/jukebox"
	"github.com/nanitefactory/visual/super"
	"github.com/sqweek/dialog"
	"golang.org/x/image/colornames"
)

func init() {
	fmt.Println() // The standard out is flushed. (This is to prevent bugs regarding windows syscalls.)
}

// -------------------------------------------------------------------------
// Itf export (Actor)

// HUD is an Actor positioned in screen-coord.
type HUD interface {
	Actor
	PosOnScreen(width, height float64) // Callback on screen resize from mainthread.
}

// Actor is what a Visualizer visualizes.
// Actor updates and draws itself. It acts as a game (virtual) object.
//
// The mainthread, called Visualizer,
// will do what's shown below, every single frame.
//
//	func (v *Visualizer) _NextFrame(dt float64) {
//		// ---------------------------------------------------
//		// 1. update - calc state of game (virtual) objects each frame
//		v._Update(dt)
//		v.fpsw.Poll()
//
//		// ---------------------------------------------------
//		// 2. draw on window
//		v.window.Clear(v.bg) // clear canvas
//		v._Draw()            // then draw
//
//		// ---------------------------------------------------
//		// 3. update window - always end with it
//		v.window.Update()
//		<-v.vsync
//	}
//
type Actor interface {
	Drawer
	Updater
}

// Drawer draws itself on a target canvas.
//
// The mainthread will do what's shown below every single frame.
//
//	// Canvas a game (virtual) world
//	t.SetMatrix(v.camera.Transform())
//
//	// For all actors, Draw() in an order.
//	for i := range v.actors {
//		v.actors[i].Draw(t)
//	}
//
type Drawer interface {
	// Draw obligatorily invoked by Visualizer on mainthread.
	Draw(t pixel.Target)
}

// Updater updates itself with the delta time given, every frame on mainthread.
//
// The mainthread will do what's shown below every single frame.
//
//	// For all actors, Update() in an order.
//	for i := range v.actors {
//		v.actors[i].Update(dt)
//	}
//
type Updater interface {
	// Update obligatorily invoked by Visualizer on mainthread.
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

// PosCenterGame returns the world center in game position.
func (c Config) PosCenterGame() pixel.Vec {
	return pixel.V(c.Width/2, c.Height/2)
}

// -------------------------------------------------------------------------
// Visualizer

// Visualizer is a mainthread that visualizes stuff.
//
// Visualizer manages:
//  1. A window
//  2. Actors; General Actors or HUDs
//  3. A game-like visualizer system along with vsync/fps/dt/camera
//
// Inputs handled by Visualizer by default: Esc, Tab, Enter, Space, Arrows, Left click, Wheeling, Ctrl+M and Ctrl+Click
//
type Visualizer struct { // also called a game
	// something system, something runtime
	window *pixelgl.Window // lazy init
	bg     pixel.RGBA
	camera *super.Camera // lazy init
	fpsw   *actors.FPSWatch
	dtw    super.DtWatch
	vsync  <-chan time.Time // lazy init
	// game (visualizer) state
	isTitleChanged bool
	// drawings
	mutex      sync.Mutex // actors must be locked up
	actors     []Actor
	huds       []HUD
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
func NewVisualizer(cfg Config, optionalHUDs []HUD, generalActors ...Actor) *Visualizer {
	if optionalHUDs == nil {
		optionalHUDs = []HUD{}
	}
	v := Visualizer{
		bg:   cfg.Bg,
		fpsw: actors.NewFPSWatchSimple(pixel.V(cfg.WinWidth, cfg.WinHeight), super.Top, super.Right),
		actors: func() []Actor { // Actors in game coords. (general actors)
			ret := make([]Actor, len(generalActors))
			for i := range generalActors {
				ret[i] = generalActors[i]
			}
			return ret
		}(),
		huds: func() []HUD { // Actors in screen coords. (HUDs)
			ret := make([]HUD, len(optionalHUDs))
			for i := range optionalHUDs {
				ret[i] = optionalHUDs[i]
			}
			return ret
		}(),
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

	// This (so-called jukebox) will be finalized (cleaned-up) when the window gets closed.
	if err := jukebox.Initialize(); err != nil {
		v.logPrintln("This function successfully returns, though there was an error: ", err)
	}

	return &v
}

// -------------------------------------------------------------------------
// Exported methods

// PushActors to this visualizer.
func (v *Visualizer) PushActors(actors ...Actor) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.actors = append(v.actors, actors...)
}

// PopActor of this visualizer.
func (v *Visualizer) PopActor() Actor {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	pop := v.actors[len(v.actors)-1]
	v.actors = v.actors[:len(v.actors)-1]
	return pop
}

// RemoveActor of this visualizer.
// Time complexity is O(N); N is the number of actors available in this visualizer.
func (v *Visualizer) RemoveActor(thisGuyGetsRemoved Actor) (removedIndeed bool) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for i, actorFound := range v.actors {
		if actorFound == thisGuyGetsRemoved {
			v.actors = append(v.actors[:i], v.actors[i+1:]...)
			return true
		}
	}
	return false
}

// PushHUDs to this visualizer. (HUD: Screen-positioned Actor.)
func (v *Visualizer) PushHUDs(actorHUDs ...HUD) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.huds = append(v.huds, actorHUDs...)
}

// PopHUD of this visualizer. (HUD: Screen-positioned Actor.)
func (v *Visualizer) PopHUD() HUD {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	pop := v.huds[len(v.huds)-1]
	v.huds = v.huds[:len(v.huds)-1]
	return pop
}

// RemoveHUD of this visualizer. (HUD: Screen-positioned Actor.)
// Time complexity is O(N); N is the number of HUDs available in this visualizer.
func (v *Visualizer) RemoveHUD(thisGuyGetsRemoved HUD) (removedIndeed bool) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	for i, actorHUDFound := range v.huds {
		if actorHUDFound == thisGuyGetsRemoved {
			v.huds = append(v.huds[:i], v.huds[i+1:]...)
			return true
		}
	}
	return false
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

// Close this visualizer. This function breaks the run loop of this.
func (v *Visualizer) Close() {
	v.window.SetClosed(true)
}

// Title gets the title and the version be displayed in the title bar.
func (v *Visualizer) Title() (fullname, title, version string) {
	fullname = v.title + "  (" + v.version + ")" // What's written or what's going to be written in the title bar.
	title = v.title
	version = v.version
	return fullname, title, version
}

// SetTitle updates the title and the version be displayed in the title bar.
func (v *Visualizer) SetTitle(title, version string) {
	if v.title == title && v.version == version { // ol the same
		return
	}
	v.isTitleChanged = true
	if v.title != title {
		v.title = title
	}
	if v.version != version {
		v.version = version
	}
}

// -------------------------------------------------------------------------
// Unexported self-updating methods - engine encapsulated

// Draw instructs this visualizer to draw its Actors.
func (v *Visualizer) _Draw() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// The target canvas Draw() draws on is called t.
	var t pixel.BasicTarget
	t = v.window

	// ---------------------------------------------------
	// 1. canvas a game world
	t.SetMatrix(v.camera.Transform())

	// Draw() all general actors in order.
	for i := range v.actors {
		v.actors[i].Draw(t)
	}

	// Default general actor gets placed after custom ones above.
	v.explosions.Draw(t)

	// Custom action after all general actors got drawn.
	if v.onDrawn != nil {
		v.onDrawn(t)
	}

	// ---------------------------------------------------
	// 2. canvas a screen
	t.SetMatrix(pixel.IM)

	// Draw()s all HUDs in an order.
	for i := range v.huds {
		v.huds[i].Draw(t)
	}

	// Default HUD.
	v.fpsw.Draw(t)
}

// Update instructs this visualizer to update its Actors.
func (v *Visualizer) _Update(dt float64) {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	// The camera would and should update every frame.
	v.camera.Update(dt)

	// All general actors Update() in order.
	for i := range v.actors {
		v.actors[i].Update(dt)
	}

	// Default general actor gets placed after custom ones above.
	v.explosions.Update(dt)

	// All HUDs Update() in order.
	for i := range v.huds {
		v.huds[i].Update(dt)
	}

	// Default HUD.
	v.fpsw.Update(dt)

	// Custom action after that all actors got updated.
	if v.onUpdated != nil {
		v.onUpdated(dt)
	}
}

func (v *Visualizer) _OnResize(width, height float64) {
	v.camera.SetScreenBound(pixel.R(0, 0, width, height))

	// Position our actors in screen coords.
	for i := range v.huds { // All huds(actors) PosOnScreen() in order.
		v.huds[i].PosOnScreen(width, height)
	}

	// Default HUD.
	v.fpsw.PosOnScreen(width, height)

	// Custom action on resized.
	if v.onResized != nil {
		v.onResized(width, height)
	}
}

// unexported because the lazy dude should be handled with care (for not guaranteeing the safety)
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

// -------------------------------------------------------------------------
// Trivial dependency

func (v *Visualizer) logPrintln(args ...interface{}) {
	if v.onLogging != nil {
		v.onLogging(args...)
	}
	log.Println(args...)
}

// -------------------------------------------------------------------------
// Read only getter method(s)

// WindowDeep is a hacky way to access `glfw.Window`.
// It returns (window *glfw.Window) which is an unexported member inside a (*pixelgl.Window).
func (v *Visualizer) _WindowDeep() (baseWindow *glfw.Window) {
	return *(**glfw.Window)(unsafe.Pointer(reflect.Indirect(reflect.ValueOf(v.window)).FieldByName("window").UnsafeAddr()))
}

// -------------------------------------------------------------------------
// Run on mainthread - EP

// Run the game window and its event loop on mainthread.
// This function must be called from the main function of an
// application so that it can work on the context of OpenGL.
func (v *Visualizer) Run() {
	pixelgl.Run(func() {
		v._RunLazyInit()
		v._RunEventLoop()
	})
}

func (v *Visualizer) _RunLazyInit() {
	// This window will show up as soon as it is created.
	win, err := pixelgl.NewWindow(pixelgl.WindowConfig{
		Title:       func(a, b, c string) string { return a }(v.Title()),
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
	v.camera = super.NewCamera(pixel.V(v.width/2, v.height/2), v.window.Bounds())

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
		txt := text.New(v.window.Bounds().Center() /* screenCenter */, AtlasASCII36())
		txt.WriteString("Loading...")
		txt.Draw(v.window, pixel.IM)
		v.window.Update()
	}
	v._NextFrame(v.dtw.Dt()) // Give it a blood pressure.
	v._NextFrame(v.dtw.Dt()) // Now the oxygenated blood will start to pump through its vein.
	// Do whatever you want after that...

	// from user setting
	v._OnResize(float64(v.winWidth), float64(v.winHeight))
	v.camera.Zoom(float64(v.initialZoomLevel))
	v.camera.Rotate(v.initialRotateDegree)
}

func (v *Visualizer) _RunEventLoop() {
	for v.window.Closed() != true { // Your average event loop in mainthread.
		// Notice that all function calls as go routine are non-blocking, but the others will block the mainthread.

		// ---------------------------------------------------
		// 0. dt
		dt := v.dtw.Dt()

		// ---------------------------------------------------
		// 1. handling events
		v._HandleEvents(dt)

		// ---------------------------------------------------
		// 2. move on
		v._NextFrame(dt)

	} // for
} // func

func (v *Visualizer) _HandleEvents(dt float64) {
	// Notice that all function calls as go routine are non-blocking, but the others will block the mainthread.

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

	// "distracting" music
	if v.window.JustReleased(pixelgl.KeyM) {
		if v.window.Pressed(pixelgl.KeyLeftControl) { // because annoying stuff
			if !jukebox.IsPlaying() {
				// The purpose of this crappy music: the music works like those beep sounds out of patient monitors.
				// When it slows down, we at least get an idea that something isn't going quite smoothly.
				jukebox.Play()
			}
		}
	}

	// click or ctrl+click
	if v.window.JustReleased(pixelgl.MouseButtonLeft) {
		posWin := v.window.MousePosition()
		posGame := v.camera.Unproject(posWin)
		go func() {
			v.explosions.ExplodeAt(pixel.V(posGame.X, posGame.Y), pixel.V(10, 10))
		}()
		if v.window.Pressed(pixelgl.KeyLeftControl) { // because annoying stuff
			// strTitle := fmt.Sprint(posGame.X, ", ", posGame.Y) //
			strDlg := fmt.Sprint(
				"camera angle in degree: ", (v.camera.Angle()/math.Pi)*180, "\r\n", "\r\n",
				"camera coordinates: ", v.camera.XY().X, v.camera.XY().Y, "\r\n", "\r\n",
				"game clock: ", v.dtw.GetTimeStarted(), "\r\n", "\r\n",
				"mouse click coords in screen pos: ", posWin.X, posWin.Y, "\r\n", "\r\n",
				"mouse click coords in game pos: ", posGame.X, posGame.Y,
			)
			go func() {
				fmt.Println() // this line resolves a syscall bug
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
	// 3. update title bar
	if v.isTitleChanged {
		v.isTitleChanged = false
		displayed, _, _ := v.Title()
		v.window.SetTitle(displayed)
	}

	// ---------------------------------------------------
	// 4. update window - always end with it
	v.window.Update()
	<-v.vsync
}
