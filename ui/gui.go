package ui

import (
	"encoding/json"
	"os"
	"strings"

	"downtomate/config"
	"downtomate/engine"
	"downtomate/watcher"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type RuleModel struct {
	walk.ListModelBase
	items []config.Rule
}

func (m *RuleModel) ItemCount() int              { return len(m.items) }
func (m *RuleModel) Value(index int) interface{} { return m.items[index].Folder }

type UI struct {
	mainWindow *walk.MainWindow
	ni         *walk.NotifyIcon
	cfg        *config.Config
	eng        *engine.Engine
	w          *watcher.Watcher
	cfgPath    string
	isRunning  bool
	appIcon    *walk.Icon

	// UI Controls
	statusLabel *walk.Label
	dirLineEdit *walk.LineEdit
	startBtn    *walk.PushButton
	stopBtn     *walk.PushButton

	ruleListBox *walk.ListBox
	ruleModel   *RuleModel

	ruleFolderEdit *walk.LineEdit
	ruleModeCombo  *walk.ComboBox

	ruleExtLabel *walk.Label
	ruleExtEdit  *walk.LineEdit

	ruleKwLabel *walk.Label
	ruleKwEdit  *walk.LineEdit

	rulePromptLabel *walk.Label
	rulePromptEdit  *walk.TextEdit

	currentRuleIdx int
}

func RunGUI(cfgPath string, cfg *config.Config, eng *engine.Engine, w *watcher.Watcher, showNow bool) error {
	ui := &UI{
		cfg:            cfg,
		eng:            eng,
		w:              w,
		cfgPath:        cfgPath,
		ruleModel:      &RuleModel{items: cfg.Rules},
		currentRuleIdx: -1,
	}

	// Load custom icon
	var err error
	ui.appIcon, err = walk.NewIconFromFile("icon.ico")
	if err != nil {
		// Fallback jika file icon.ico dihapus secara tidak sengaja oleh user saat runtime
		ui.appIcon, _ = walk.NewIconFromResourceId(2)
	}

	err = MainWindow{
		AssignTo: &ui.mainWindow,
		Title:    "DownToMate - Control Panel",
		Icon:     ui.appIcon,
		MinSize:  Size{Width: 700, Height: 550},
		Size:     Size{Width: 700, Height: 550},
		Layout:   VBox{},
		Children: []Widget{
			GroupBox{
				Title:  "Status & Pengaturan Utama",
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Status Pantauan:"},
					Label{AssignTo: &ui.statusLabel, Text: "🔴 BERHENTI", Font: Font{Bold: true, PointSize: 12}},

					Label{Text: "Folder Target:"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							LineEdit{
								AssignTo: &ui.dirLineEdit,
								Text:     cfg.WatchDirectory,
							},
							PushButton{
								Text: "📂 Pilih Folder...",
								Font: Font{Bold: true},
								OnClicked: func() {
									dlg := new(walk.FileDialog)
									dlg.Title = "Pilih Folder Target"
									if ok, _ := dlg.ShowBrowseFolder(ui.mainWindow); ok {
										ui.dirLineEdit.SetText(dlg.FilePath)
										ui.saveConfigToDisk()
									}
								},
							},
						},
					},
					Composite{
						Layout:     HBox{MarginsZero: true},
						ColumnSpan: 2,
						Children: []Widget{
							PushButton{
								AssignTo:  &ui.startBtn,
								Text:      "▶️ MULAI PANTAU",
								Font:      Font{Bold: true, PointSize: 10},
								MinSize:   Size{Width: 140, Height: 35},
								OnClicked: ui.startEngine,
							},
							PushButton{
								AssignTo:  &ui.stopBtn,
								Text:      "⏹️ HENTIKAN",
								Font:      Font{Bold: true, PointSize: 10},
								MinSize:   Size{Width: 120, Height: 35},
								Enabled:   false,
								OnClicked: ui.stopEngine,
							},
							HSpacer{},
							PushButton{
								Text:    "❌ KELUAR APLIKASI (MATI TOTAL)",
								Font:    Font{Bold: true, PointSize: 9},
								MinSize: Size{Width: 200, Height: 35},
								OnClicked: func() {
									ui.ni.Dispose()
									walk.App().Exit(0)
								},
							},
						},
					},
				},
			},
			GroupBox{
				Title:  "Manajemen Aturan (Rules)",
				Layout: HBox{},
				Children: []Widget{
					ListBox{
						AssignTo:              &ui.ruleListBox,
						Model:                 ui.ruleModel,
						OnCurrentIndexChanged: ui.onRuleSelected,
						OnMouseUp: func(x, y int, button walk.MouseButton) {
							if button == walk.LeftButton {
								ui.onRuleSelected()
							}
						},
						MinSize: Size{Width: 160, Height: 0},
					},
					Composite{
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Nama Folder Tujuan:"},
							LineEdit{AssignTo: &ui.ruleFolderEdit},

							Label{Text: "Mode Pemantauan:"},
							ComboBox{
								AssignTo:              &ui.ruleModeCombo,
								Model:                 []string{"Ekstensi", "Keyword", "AI"},
								OnCurrentIndexChanged: ui.onModeChanged,
							},

							Label{AssignTo: &ui.ruleExtLabel, Text: "Ekstensi yang dipindahkan (cth: .jpg, .png):"},
							LineEdit{AssignTo: &ui.ruleExtEdit},

							Label{AssignTo: &ui.ruleKwLabel, Text: "Kata Kunci (Nama File, pisahkan dgn koma):"},
							LineEdit{AssignTo: &ui.ruleKwEdit},

							Label{AssignTo: &ui.rulePromptLabel, Text: "Instruksi AI (Isi File):"},
							TextEdit{AssignTo: &ui.rulePromptEdit, VScroll: true, MinSize: Size{Height: 60}},

							VSpacer{},
							Composite{
								Layout: HBox{MarginsZero: true},
								Children: []Widget{
									PushButton{
										Text:      "💾 SIMPAN",
										Font:      Font{Bold: true},
										MinSize:   Size{Width: 100, Height: 30},
										OnClicked: ui.saveRule,
									},
									PushButton{
										Text:      "➕ BARU",
										Font:      Font{Bold: true},
										MinSize:   Size{Width: 100, Height: 30},
										OnClicked: ui.newRule,
									},
									PushButton{
										Text:      "❌ HAPUS",
										Font:      Font{Bold: true},
										MinSize:   Size{Width: 100, Height: 30},
										OnClicked: ui.deleteRule,
									},
								},
							},
						},
					},
				},
			},
		},
	}.Create()

	if err != nil {
		return err
	}

	ui.mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if reason == walk.CloseReasonUnknown {
			*canceled = true
			ui.mainWindow.Hide()
			if ui.ni != nil {
				// Gunakan ui.appIcon untuk notifikasi Tray agar logonya sama
				ui.ni.ShowCustom("DownToMate", "Aplikasi berjalan di background. Klik ikon ini untuk membuka Control Panel.", ui.appIcon)
			}
		}
	})

	if err := ui.setupTray(); err != nil {
		return err
	}
	defer ui.ni.Dispose()

	// Initial selection
	if len(ui.ruleModel.items) > 0 {
		ui.ruleListBox.SetCurrentIndex(0)
		ui.onRuleSelected()
	}

	if showNow {
		ui.mainWindow.Show()
	}

	ui.startEngine()

	ui.mainWindow.Run()
	return nil
}

func (ui *UI) onModeChanged() {
	if ui.ruleModeCombo == nil {
		return
	}

	idx := ui.ruleModeCombo.CurrentIndex()
	var mode string
	if idx >= 0 && idx <= 2 {
		mode = []string{"Ekstensi", "Keyword", "AI"}[idx]
	} else {
		return
	}

	if ui.ruleExtLabel != nil {
		if mode == "Ekstensi" {
			ui.ruleExtLabel.SetText("Ekstensi yang dipindahkan (cth: .jpg, .png):")
		} else {
			ui.ruleExtLabel.SetText("Batasi pencarian hanya pada ekstensi (opsional):")
		}
	}

	showKw := mode == "Keyword"
	if ui.ruleKwLabel != nil {
		ui.ruleKwLabel.SetVisible(showKw)
	}
	if ui.ruleKwEdit != nil {
		ui.ruleKwEdit.SetVisible(showKw)
	}

	showAI := mode == "AI"
	if ui.rulePromptLabel != nil {
		ui.rulePromptLabel.SetVisible(showAI)
	}
	if ui.rulePromptEdit != nil {
		ui.rulePromptEdit.SetVisible(showAI)
	}
}

func (ui *UI) startEngine() {
	if ui.isRunning {
		return
	}
	ui.saveConfigToDisk()
	ui.eng.UpdateConfig(ui.cfg)
	ui.w.UpdateConfig(ui.cfg)

	ui.w.SetPaused(false)

	ui.statusLabel.SetText("🟢 BERJALAN")
	ui.startBtn.SetEnabled(false)
	ui.stopBtn.SetEnabled(true)
	ui.isRunning = true
}

func (ui *UI) stopEngine() {
	if !ui.isRunning {
		return
	}
	ui.w.SetPaused(true)

	ui.statusLabel.SetText("🔴 BERHENTI")
	ui.startBtn.SetEnabled(true)
	ui.stopBtn.SetEnabled(false)
	ui.isRunning = false
}

func (ui *UI) loadRuleDetails(index int) {
	if index < 0 || index >= len(ui.ruleModel.items) {
		ui.currentRuleIdx = -1
		ui.ruleFolderEdit.SetText("")
		ui.ruleExtEdit.SetText("")
		ui.ruleKwEdit.SetText("")
		ui.rulePromptEdit.SetText("")
		return
	}

	ui.currentRuleIdx = index
	rule := ui.ruleModel.items[index]
	ui.ruleFolderEdit.SetText(rule.Folder)

	mode := rule.Mode
	if mode == "" {
		mode = "Ekstensi"
	}

	switch mode {
	case "Ekstensi":
		ui.ruleModeCombo.SetCurrentIndex(0)
	case "Keyword":
		ui.ruleModeCombo.SetCurrentIndex(1)
	case "AI":
		ui.ruleModeCombo.SetCurrentIndex(2)
	}

	ui.ruleExtEdit.SetText(strings.Join(rule.Extensions, ", "))

	kws := []string{}
	for _, kw := range rule.Keywords {
		kws = append(kws, kw.Word)
	}
	ui.ruleKwEdit.SetText(strings.Join(kws, ", "))
	ui.rulePromptEdit.SetText(rule.AIPrompt)

	ui.onModeChanged()
}

func (ui *UI) onRuleSelected() {
	index := ui.ruleListBox.CurrentIndex()
	if ui.mainWindow != nil {
		ui.mainWindow.Synchronize(func() {
			ui.loadRuleDetails(index)
		})
		return
	}
	ui.loadRuleDetails(index)
}

func (ui *UI) refreshRuleListSelection(index int) {
	ui.ruleListBox.SetModel(nil)
	ui.ruleListBox.SetModel(ui.ruleModel)
	ui.ruleListBox.SetCurrentIndex(index)
	if ui.mainWindow != nil {
		ui.mainWindow.Synchronize(func() {
			ui.loadRuleDetails(index)
		})
		return
	}
	ui.loadRuleDetails(index)
}

func (ui *UI) saveRule() {
	idx := ui.currentRuleIdx
	if idx < 0 {
		return
	}
	rule := &ui.ruleModel.items[idx]
	rule.Folder = ui.ruleFolderEdit.Text()
	rule.Mode = ui.ruleModeCombo.Text()

	exts := strings.Split(ui.ruleExtEdit.Text(), ",")
	var cleanExts []string
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e != "" {
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			cleanExts = append(cleanExts, strings.ToLower(e))
		}
	}
	rule.Extensions = cleanExts

	kws := strings.Split(ui.ruleKwEdit.Text(), ",")
	var cleanKws []config.Keyword
	for _, kw := range kws {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			cleanKws = append(cleanKws, config.Keyword{Word: kw, Weight: 5})
		}
	}
	rule.Keywords = cleanKws
	if len(cleanKws) > 0 {
		rule.KeywordThreshold = 5
	} else {
		rule.KeywordThreshold = 0
	}

	rule.AIPrompt = ui.rulePromptEdit.Text()

	ui.refreshRuleListSelection(idx)

	ui.saveConfigToDisk()
	walk.MsgBox(ui.mainWindow, "Info", "Aturan berhasil disimpan!", walk.MsgBoxIconInformation)
}

func (ui *UI) newRule() {
	newRule := config.Rule{
		Folder:     "Folder_Baru",
		Mode:       "Ekstensi",
		Extensions: []string{".xyz"},
	}
	ui.ruleModel.items = append(ui.ruleModel.items, newRule)

	ui.refreshRuleListSelection(len(ui.ruleModel.items) - 1)

	ui.saveConfigToDisk()
}

func (ui *UI) deleteRule() {
	idx := ui.currentRuleIdx
	if idx < 0 || idx >= len(ui.ruleModel.items) {
		return
	}

	ui.ruleModel.items = append(ui.ruleModel.items[:idx], ui.ruleModel.items[idx+1:]...)

	if len(ui.ruleModel.items) > 0 {
		ui.refreshRuleListSelection(0)
	} else {
		ui.ruleListBox.SetModel(nil)
		ui.ruleListBox.SetModel(ui.ruleModel)
		ui.loadRuleDetails(-1)
	}

	ui.saveConfigToDisk()
}

func (ui *UI) saveConfigToDisk() {
	ui.cfg.WatchDirectory = ui.dirLineEdit.Text()
	ui.cfg.Rules = ui.ruleModel.items
	data, _ := json.MarshalIndent(ui.cfg, "", "  ")
	os.WriteFile(ui.cfgPath, data, 0644)
	ui.eng.UpdateConfig(ui.cfg)
	ui.w.UpdateConfig(ui.cfg)
}

func (ui *UI) setupTray() error {
	var err error
	ui.ni, err = walk.NewNotifyIcon(ui.mainWindow)
	if err != nil {
		return err
	}

	if ui.appIcon != nil {
		ui.ni.SetIcon(ui.appIcon)
	}

	ui.ni.SetToolTip("DownToMate Control Panel")
	ui.ni.SetVisible(true)

	ui.ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if !ui.mainWindow.Visible() {
				ui.mainWindow.Show()
			} else {
				ui.mainWindow.Hide()
			}
		}
	})

	menuActionOpen := walk.NewAction()
	menuActionOpen.SetText("Buka Control Panel")
	menuActionOpen.Triggered().Attach(func() {
		ui.mainWindow.Show()
	})
	ui.ni.ContextMenu().Actions().Add(menuActionOpen)

	menuActionExit := walk.NewAction()
	menuActionExit.SetText("Keluar Sepenuhnya")
	menuActionExit.Triggered().Attach(func() {
		ui.ni.Dispose()
		walk.App().Exit(0)
	})
	ui.ni.ContextMenu().Actions().Add(menuActionExit)

	return nil
}
