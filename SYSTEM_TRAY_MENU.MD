System Tray Menus
​
 Summarise
​
Wails provides unified system tray APIs that work across all platforms. Create tray icons with menus, attach windows, and handle clicks with native platform behaviour for background applications, services, and quick-access utilities.

    "github.com/wailsapp/wails/v3/pkg/application"
//go:embed assets/icon.png
    app := application.New(application.Options{
        Name: "Tray App",
    // Create system tray
    systray := app.SystemTray.New()
    systray.SetIcon(icon)
    systray.SetLabel("My App")
    menu := app.NewMenu()
    menu.Add("Show").OnClick(func(ctx *application.Context) {
        // Show main window
    menu.Add("Quit").OnClick(func(ctx *application.Context) {
    systray.SetMenu(menu)
    // Create hidden window
    window := app.Window.New()
Result: System tray icon with menu on all platforms.

systray := app.SystemTray.New()
systray.SetIcon(iconBytes)
// Set label (macOS) / tooltip (Windows)
systray.SetLabel("My Application")
Icons should be embedded:

//go:embed assets/icon.png
//go:embed assets/icon-dark.png
    app := application.New(application.Options{
    systray := app.SystemTray.New()
    systray.SetIcon(icon)
    systray.SetDarkModeIcon(iconDark)  // macOS dark mode
Icon requirements:

Platform	Size	Format	Notes
Windows	16x16 or 32x32	PNG, ICO	Notification area
macOS	18x18 to 22x22	PNG	Menu bar, template recommended
Linux	22x22 to 48x48	PNG, SVG	Varies by DE
Template icons adapt to light/dark mode automatically:

systray.SetTemplateIcon(iconBytes)
Template icon guidelines:

Use black and clear (transparent) colours only
Black becomes white in dark mode
Name file with Template suffix: iconTemplate.png
Design guide
System tray menus work like application menus:

menu.Add("Open").OnClick(func(ctx *application.Context) {
menu.AddCheckbox("Start at Login", false).OnClick(func(ctx *application.Context) {
    enabled := ctx.ClickedMenuItem().Checked()
    setStartAtLogin(enabled)
menu.Add("Quit").OnClick(func(ctx *application.Context) {
For all menu item types, see Menu Reference.

Attach a window to the tray icon for automatic show/hide:

window := app.Window.New()
systray.AttachWindow(window)
systray.SetWindowOffset(10)  // Pixels from tray icon
systray.SetWindowDebounce(200 * time.Millisecond)  // Click debounce
Behaviour:

Window starts hidden
Left-click tray icon → Toggle window visibility
Right-click tray icon → Show menu (if set)
Window positioned near tray icon
Example: Popup window

window := app.Window.NewWithOptions(application.WebviewWindowOptions{
    Title:  "Quick Access",
    Frameless: true,  // No title bar
    AlwaysOnTop: true,  // Stay on top
systray.AttachWindow(window)
systray.SetWindowOffset(5)
Handle tray icon clicks:

systray := app.SystemTray.New()
    fmt.Println("Tray icon clicked")
systray.OnRightClick(func() {
    fmt.Println("Tray icon right-clicked")
systray.OnDoubleClick(func() {
    fmt.Println("Tray icon double-clicked")
systray.OnMouseEnter(func() {
    fmt.Println("Mouse entered tray icon")
systray.OnMouseLeave(func() {
    fmt.Println("Mouse left tray icon")
Platform support:

Event	Windows	macOS	Linux
OnClick	✅	✅	✅
OnRightClick	✅	✅	✅
OnDoubleClick	✅	✅	⚠️ Varies
OnMouseEnter	✅	✅	⚠️ Varies
OnMouseLeave	✅	✅	⚠️ Varies
Update tray icon and menu dynamically:

        systray.SetIcon(activeIcon)
        systray.SetLabel("Active")
        systray.SetIcon(inactiveIcon)
        systray.SetLabel("Inactive")
pauseMenuItem := menu.Add("Pause")
pauseMenuItem.OnClick(func(ctx *application.Context) {
        pauseMenuItem.SetLabel("Resume")
        pauseMenuItem.SetLabel("Pause")
    menu.Update()  // Important!
Always Call Update()

After changing menu state, call menu.Update(). See Menu Reference.

For major changes, rebuild the entire menu:

func rebuildTrayMenu(status string) {
    menu := app.NewMenu()
    // Status-specific items
        menu.Add("Syncing...").SetEnabled(false)
        menu.Add("Pause Sync").OnClick(pauseSync)
        menu.Add("Up to date ✓").SetEnabled(false)
        menu.Add("Sync Now").OnClick(startSync)
        menu.Add("Sync Error").SetEnabled(false)
        menu.Add("Retry").OnClick(retrySync)
    menu.Add("Quit").OnClick(func(ctx *application.Context) {
    systray.SetMenu(menu)
Platform-Specific Features

Menu bar integration:

// Set label (appears next to icon)
systray.SetLabel("My App")
// Use template icon (adapts to dark mode)
systray.SetTemplateIcon(iconBytes)
systray.SetIconPosition(application.IconPositionRight)
Icon positions:

IconPositionLeft - Icon left of label
IconPositionRight - Icon right of label
IconPositionOnly - Icon only, no label
IconPositionNone - Label only, no icon
Best practices:

Use template icons (black + transparent)
Keep labels short (3-5 characters)
18x18 to 22x22 pixels for Retina displays
Test in both light and dark modes
Here’s a production-ready system tray application:

    "github.com/wailsapp/wails/v3/pkg/application"
//go:embed assets/icon.png
//go:embed assets/icon-active.png
    app     *application.Application
    systray *application.SystemTray
    window  *application.WebviewWindow
    menu    *application.Menu
    app := application.New(application.Options{
        Name: "Tray Application",
        Mac: application.MacOptions{
            ApplicationShouldTerminateAfterLastWindowClosed: false,
    trayApp := &TrayApp{app: app}
func (t *TrayApp) setup() {
    // Create system tray
    t.systray = t.app.SystemTray.New()
    t.systray.SetIcon(icon)
    t.systray.SetLabel("Inactive")
    // Create window (hidden by default)
    t.window = t.app.Window.NewWithOptions(application.WebviewWindowOptions{
        Title:  "Tray Application",
    // Attach window to tray
    t.systray.AttachWindow(t.window)
    t.systray.SetWindowOffset(10)
    // Handle tray clicks
    t.systray.OnRightClick(func() {
        t.systray.OpenMenu()
    // Start background task
    go t.backgroundTask()
func (t *TrayApp) createMenu() {
    t.menu = t.app.NewMenu()
    // Status item (disabled)
    statusItem := t.menu.Add("Status: Inactive")
    statusItem.SetEnabled(false)
    t.menu.AddSeparator()
    t.menu.Add("Start").OnClick(func(ctx *application.Context) {
    t.menu.Add("Show Window").OnClick(func(ctx *application.Context) {
        t.window.SetFocus()
    t.menu.AddSeparator()
    t.menu.AddCheckbox("Start at Login", false).OnClick(func(ctx *application.Context) {
        enabled := ctx.ClickedMenuItem().Checked()
        t.setStartAtLogin(enabled)
    t.menu.AddSeparator()
    t.menu.Add("Quit").OnClick(func(ctx *application.Context) {
    t.systray.SetMenu(t.menu)
func (t *TrayApp) toggleActive() {
    t.isActive = !t.isActive
func (t *TrayApp) updateTray() {
        t.systray.SetIcon(iconActive)
        t.systray.SetLabel("Active")
        t.systray.SetIcon(icon)
        t.systray.SetLabel("Inactive")
    // Rebuild menu with new status
func (t *TrayApp) backgroundTask() {
    ticker := time.NewTicker(5 * time.Second)
            fmt.Println("Background task running...")
func (t *TrayApp) setStartAtLogin(enabled bool) {
    // Implementation varies by platform
    fmt.Printf("Start at login: %v\n", enabled)
Show/hide the tray icon dynamically:

    fmt.Println("Tray icon is visible")
Platform Support:

Platform	Hide()	Show()	Notes
Windows	✅	✅	Fully functional - icon appears/disappears from notification area
macOS	✅	✅	Menu bar item shows/hides
Linux	✅	✅	Varies by desktop environment
Use cases:

Temporarily hide tray icon based on user preference
Headless mode with tray icon appearing only when needed
Toggle visibility based on application state
Example - Conditional Tray Visibility:

func (t *TrayApp) setTrayVisibility(visible bool) {
// Show tray only when updates are available
func (t *TrayApp) checkForUpdates() {
        t.systray.SetLabel("Update Available")
Destroy the tray icon when done:

app := application.New(application.Options{
        if systray != nil {
            systray.Destroy()
Important: Always destroy system tray on shutdown to release resources.

Use template icons on macOS - Adapts to dark mode
Keep labels short - 3-5 characters maximum
Provide tooltips on Windows - Helps users identify your app
Test on all platforms - Behaviour varies
Handle clicks appropriately - Left-click for main action, right-click for menu
Update icon for status - Visual feedback is important
Destroy on shutdown - Release resources
Don’t use large icons - Follow platform guidelines
Don’t use long labels - Gets truncated
Don’t forget dark mode - Test on macOS dark mode
Don’t block click handlers - Keep them fast
Don’t forget menu.Update() - After changing menu state
Don’t assume tray support - Some Linux DEs don’t support it
Possible causes:

Icon format not supported
Icon size too large/small
System tray not supported (Linux)
Solution:

// Check if system tray is supported
if !application.SystemTraySupported() {
    fmt.Println("System tray not supported")
    // Fallback to window-only mode
Icon Looks Wrong on macOS

Cause: Not using template icon

Solution:

systray.SetTemplateIcon(iconBytes)
// Or design icon as template (black + transparent)
Cause: Forgot to call menu.Update()

Solution:

menuItem.SetLabel("New Label")
menu.Update()  // Add this!
Menu Reference

Complete reference for menu item types and properties.

Learn More →

Application Menus

Create application menu bars.

Learn More →

Context Menus

Create right-click context menus.

Learn More →

System Tray Tutorial

Build a complete system tray application.

Learn More →

Questions? Ask in Discord or check the system tray examples.