Migrating from v2 to v3
​
 Summarise
​
Wails v3 is a complete rewrite with significant improvements in architecture, performance, and developer experience. This guide helps you migrate your v2 application to v3.

Key changes:

New application structure
Improved bindings system
Enhanced window management
Better event system
Simplified configuration
Migration time: 1-4 hours for typical applications

Application Initialisation

In v2, application setup, window configuration, and execution were all combined into a single wails.Run() call. This monolithic approach made it difficult to create multiple windows, handle errors at different stages, or test individual components of your application.

v3 separates these concerns into distinct phases: application creation, window creation, and execution. This separation gives you explicit control over each stage of your application’s lifecycle and makes the code more modular and testable.

v2:

err := wails.Run(&options.App{
v3:

app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&GreetService{}),
window := app.Window.NewWithOptions(application.WebviewWindowOptions{
Why this is better:

Multi-window support: You can create windows dynamically at any point, not just at startup
Better error handling: Each phase can be validated separately with proper error handling
Clearer code: The separation makes it obvious what’s happening at each stage
More testable: You can test application setup without running the event loop
More flexible: Windows can be created, destroyed, and recreated throughout the application lifecycle
In v2, every bound struct required a context field and a startup(ctx) method to receive the runtime context. This created tight coupling between your business logic and the Wails runtime, making code harder to test and understand.

v3 introduces the service pattern, where your structs are completely standalone and don’t need to store runtime context. If a service needs access to the application instance, it explicitly receives it through dependency injection rather than implicit context threading.

v2:

func (a *App) startup(ctx context.Context) {
func (a *App) Greet(name string) string {
    return "Hello " + name
v3:

type GreetService struct{}
func (g *GreetService) Greet(name string) string {
    return "Hello " + name
app := application.New(application.Options{
    Services: []application.Service{
        application.NewService(&GreetService{}),
Why this is better:

No implicit dependencies: Services are plain Go structs without hidden runtime dependencies
Easier testing: You can test service methods without mocking a Wails context
Clearer code: Dependencies are explicit (passed as constructor arguments) rather than hidden in a context field
Better organization: Services can be grouped by domain rather than all living in a single App struct
Proper initialization: Use ServiceStartup() method when you need initialization, making it explicit
In v2, all runtime operations required passing a context to global functions from the runtime package. This created tight coupling to the context object throughout your codebase and made the API feel procedural rather than object-oriented.

v3 replaces the context-based runtime with direct method calls on application and window objects. Operations are called directly on the objects they affect, making the code more intuitive and object-oriented.

v2:

import "github.com/wailsapp/wails/v2/pkg/runtime"
runtime.WindowSetTitle(a.ctx, "New Title")
runtime.EventsEmit(a.ctx, "event-name", data)
v3:

    app *application.Application
func (s *MyService) UpdateTitle() {
    window := s.app.Window.Current()
    window.SetTitle("New Title")
func (s *MyService) EmitEvent() {
    s.app.Event.Emit("event-name", data)
Why this is better:

Object-oriented design: Methods are called on the objects they affect (window, app, menu, etc.)
Clearer intent: window.SetTitle() is more obvious than runtime.WindowSetTitle(ctx, ...)
Better IDE support: Autocomplete works properly when methods are on objects
Multi-window clarity: With multiple windows, you explicitly choose which window to operate on
No context threading: You don’t need to pass context through every function
In v2, bindings were organized by Go package and struct name, typically resulting in paths like wailsjs/go/main/App. This structure didn’t reflect logical grouping and made it hard to find related functionality.

v3 organizes bindings by service name and application module, creating a clearer logical structure. The bindings are generated into a bindings directory organized by your application name and service names, making it easier to understand what functionality is available.

v2:

import { Greet } from '../wailsjs/go/main/App'
const result = await Greet("World")
v3:

import { Greet } from './bindings/myapp/greetservice'
const result = await Greet("World")
Why this is better:

Logical organization: Bindings are grouped by service name rather than Go package structure
Clearer imports: The path reflects the domain logic (greetservice) not the file structure (main/App)
Better discoverability: You can navigate bindings by feature rather than by technical structure
Consistent naming: Service-based organization matches your backend architecture
Simpler paths: No more ../wailsjs/go prefix - just ./bindings
In v2, events used variadic interface{} parameters and required passing context to every event function. Event handlers received untyped data that needed manual type assertions, making the event system error-prone and hard to debug.

v3 introduces typed event objects and removes the context requirement. Event handlers receive a proper event object with typed data, making the event system more reliable and easier to use.

v2:

runtime.EventsOn(ctx, "event-name", func(data ...interface{}) {
runtime.EventsEmit(ctx, "event-name", data)
v3:

app.Event.On("event-name", func(e *application.CustomEvent) {
app.Event.Emit("event-name", data)
Why this is better:

Type safety: Events use proper event objects instead of ...interface{}
Better debugging: Event objects contain metadata like event name, making debugging easier
Clearer API: app.Event.On() and app.Event.Emit() are more intuitive than runtime functions
No context needed: Events work directly on the app object without threading context
Simpler handlers: Event handlers have a clear signature instead of variadic parameters
v2 supported only a single window per application. The window was created at startup and all window operations were performed through runtime functions that implicitly targeted that single window.

v3 introduces native multi-window support as a core feature. Each window is a first-class object with its own methods and lifecycle. You can create, manage, and destroy multiple windows dynamically throughout your application’s lifetime.

v2:

runtime.WindowSetSize(ctx, 800, 600)
v3:

// Multiple windows supported
window1 := app.Window.New()
window1.SetSize(800, 600)
window2 := app.Window.New()
window2.SetSize(1024, 768)
Why this is better:

Multi-window applications: Build apps with multiple independent windows (dashboards, preferences, tools, etc.)
Explicit window references: Each window is an object you can store and manipulate directly
Dynamic window creation: Create and destroy windows at any time during runtime
Independent window state: Each window has its own events, properties, and lifecycle
Better architecture: Window management is object-oriented rather than context-based
Step 1: Update Dependencies

go.mod:

    github.com/wailsapp/wails/v3 v3.0.0-alpha.1
Update:

go get github.com/wailsapp/wails/v3@latest
v2:

    "github.com/wailsapp/wails/v2/pkg/options"
    "github.com/wailsapp/wails/v2/pkg/options/assetserver"
    "github.com/wailsapp/wails/v2/pkg/options/windows"
//go:embed all:frontend/dist
    err := wails.Run(&options.App{
        Title:  "My App",
        AssetServer: &assetserver.Options{
            Assets: assets,
        Bind: []interface{}{
        Windows: &windows.Options{
            WebviewIsTransparent: false,
        println("Error:", err.Error())
v3:

    "github.com/wailsapp/wails/v3/pkg/application"
    app := application.New(application.Options{
        Services: []application.Service{
            application.NewService(&MyService{}),
        Assets: application.AssetOptions{
            Handler: application.AssetFileServerFS(assets),
    app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "My App",
Step 3: Convert App Struct to Service

v2:

func (a *App) startup(ctx context.Context) {
func (a *App) Greet(name string) string {
    return "Hello " + name
v3:

    app *application.Application
func NewMyService(app *application.Application) *MyService {
    return &MyService{app: app}
func (s *MyService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
func (s *MyService) Greet(name string) string {
    return "Hello " + name
// Register after app creation
app := application.New(application.Options{})
app.RegisterService(application.NewService(NewMyService(app)))
Step 4: Update Runtime Calls

v2:

func (a *App) DoSomething() {
    runtime.WindowSetTitle(a.ctx, "New Title")
    runtime.EventsEmit(a.ctx, "update", data)
    runtime.LogInfo(a.ctx, "Message")
v3:

func (s *MyService) DoSomething() {
    window := s.app.Window.Current()
    window.SetTitle("New Title")
    s.app.Event.Emit("update", data)
    s.app.Logger.Info("Message")
Generate new bindings:

Update imports:

import { Greet } from '../wailsjs/go/main/App'
import { Greet } from './bindings/myapp/myservice'
Update event handling:

import { EventsOn, EventsEmit } from '../wailsjs/runtime/runtime'
EventsOn("update", (data) => {
EventsEmit("action", data)
import { OnEvent, Emit } from '@wailsio/runtime'
OnEvent("update", (data) => {
Step 6: Update Configuration

v2 (wails.json):

  "outputfilename": "myapp",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto"
v3 (wails.json):

    "install": "npm install",
    "build": "npm run build",
    "dev": "npm run dev",
    "devServerUrl": "http://localhost:5173"
v2:

selection, err := runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{
    Title: "Select File",
v3:

selection, err := app.Dialog.OpenFile(application.OpenFileDialogOptions{
    Title: "Select File",
v2:

menu.Append(menu.Text("File", nil, []*menu.MenuItem{
    menu.Text("Quit", nil, func(_ *menu.CallbackData) {
        runtime.Quit(ctx)
v3:

fileMenu := menu.AddSubmenu("File")
fileMenu.Add("Quit").OnClick(func(ctx *application.Context) {
v2:

v3:

systray := app.SystemTray.New()
systray.SetIcon(iconBytes)
systray.SetLabel("My App")
menu.Add("Show").OnClick(showWindow)
menu.Add("Quit").OnClick(app.Quit)
Issue: Bindings not found

Problem: Import errors after migration

Solution:

Problem: ctx not available

Solution:

Store app reference instead:

    app *application.Application
func NewMyService(app *application.Application) *MyService {
    return &MyService{app: app}
Issue: Window methods not working

Problem: runtime.WindowSetTitle() doesn’t exist

Solution:

Use window methods directly:

window := s.app.Window.Current()
window.SetTitle("New Title")
Problem: Events registered but not received

Solution:

Check event names match exactly:

app.Event.Emit("my-event", data)
OnEvent("my-event", handler)  // Must match exactly
 Application starts without errors
 All bindings work
 Events are sent and received
 Windows open and close correctly
 Menus work (if applicable)
 dialogs work (if applicable)
 System tray works (if applicable)
 Build process works
 Production build works
Faster startup - Optimised initialisation
Lower memory - Efficient resource usage
Better bridge - <1ms call overhead
Multi-window - Native support
System tray - Built-in
Better events - Typed, simpler API
Services - Better code organisation
Type safety - Full TypeScript support
Better errors - Clear error messages
Hot reload - Faster development
Better docs - Comprehensive guides
Documentation
Discord Community
GitHub Issues
Examples
Q: Can I run v2 and v3 side by side?
A: Yes, they use different import paths.

Q: Is v3 production-ready?
A: v3 is in alpha/beta. Test thoroughly before production.

Q: Will v2 be maintained?
A: Yes, v2 will receive critical updates.

Q: How long does migration take?
A: 1-4 hours for typical applications.

Quick Start

Get started with Wails v3.

Learn More →

Core Concepts

Understand v3 architecture.

Learn More →

Bindings

Learn the new bindings system.

Learn More →

Examples

See complete v3 examples.

View Examples →

Questions? Ask in Discord or open an issue.