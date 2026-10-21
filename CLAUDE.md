# CLAUDE.md

查詢或強制設定華碩注音中／英模式的 Windows 小工具（Go）。先讀 `README.md`。用繁體中文回覆使用者。

## 背景：華碩注音的行為（都是實測得出，不是文件寫的）

- **IMM conversion mode（`IMC_GET/SETCONVERSIONMODE`）完全無效。** 寫入後讀回來的值會跟寫入的一樣，但華碩注音根本沒改變。第一版就是被這個假象騙了，誤以為切換成功。
- **真正的狀態在 ini**：`%ProgramFiles%\ASUS\AsusIME\zhtw\data\%UserName%\AsusIMEConfig.ini` 的 `[DirectOutputMode]`。key 是程式名稱，`0` 是中、`1` 是英；沒有記錄的程式預設是中。不管是按 Shift 還是外部觸發的切換，都會在數百 ms 內寫入。
- **open status 一改變就翻轉中英**（`IMC_SETOPENSTATUS`）。寫入的值不代表中或英，寫入跟目前相同的值則不會觸發。
- 華碩注音的 TSF CLSID 是 `{D9012138-88BF-46F4-B599-E69D8BA2FED6}`。DLL 裡有 `KEYBOARD_OPENCLOSE` compartment，但沒有 `INPUTMODE_CONVERSION`。
- **焦點所在的控制項要有 input context 才能切換。** 輸入框一定有；其他控制項看程式，一般 Win32 視窗預設會帶 context，所以使用者實測過焦點不在輸入框也常常能成功。沒有 context 的控制項（例如 WinForms 的按鈕、部分網頁本文）收不到 open status，翻轉沒有作用，只能以 exit code 5 結束。外部程式無法查詢別的程式有沒有 context，所以「500 ms 內 ini 沒變」是唯一的判斷依據。焦點移到沒有 context 的控制項時，open status 若從 1 變成 0，華碩注音會自行翻轉成英文並寫進 ini，回到輸入框再翻轉回來；它恢復時用的是記憶體裡的狀態，**直接改 ini 會被覆蓋**（實測過）。使用者只在輸入時才切換，所以不做等待焦點的功能。
- 工作列輸入指示器的 UIA 名稱也會反映真正的狀態，例如「系統匣輸入指示器 注音/英文混合模式」／「英文模式」。指示器的文字語言會跟著目標程式的介面語言改變。
- 有很多現成工具（im-select、im-select-mspy、im-control、InputTip、AutoHotkey IME.ahk）。它們靠的是 conversion mode、模擬按鍵或 TSF compartment 注入，對華碩注音不是無效就是不可靠。

## 檔案與指令

- `main.go`：全部的程式。`isChineseMode` 讀 ini，`checkOnlyAsusChineseIme` 確認登錄檔中啟用的中文輸入法都是華碩注音。
- 所有指令都透過 `task`（`task --list`）執行；`package.json` 只放 Node 工具（`prepare`、`spell`）。
  - `task build`：輸出到 `~/.local/bin`，產生 `asusimemode.exe`（主控台版）和 `asusimemodew.exe`（GUI 版，`-H=windowsgui`，給快速鍵用）。
  - `task fmt`、`task lint`：golangci-lint，設定在 `.golangci.yaml`。
  - `task spell`：cspell。新的專有名詞加進 `cspell.config.yaml`。
  - `task release:snapshot`：GoReleaser 只產生 `dist/`，不發布。
  - `task release VERSION=X.Y.Z`：檢查後建立有簽章的 `vX.Y.Z` tag 並推送，會觸發 `release.yaml`。**這是對外發布，只有使用者明確要求時才執行。**
- **要同步的地方：**
  - ldflags 在 `Taskfile.yaml` 和 `.goreleaser.yaml` 各一份。
  - GoReleaser 的版本在 `package.json`（精確版本）和 `.github/workflows/release.yaml` 各一份。
  - golangci-lint 的版本：`release.yaml` 固定成跟本機相同的版本（目前 v2.15.0）。
  - `README.md`（繁中，主要）和 `README.en.md` 內容要一致。

## commit message

嚴格的 Conventional Commits，type 照 Angular convention。husky 的 `commit-msg` hook 會用 commitlint（`.commitlintrc.yaml`）檢查。

- 英文。主旨用祈使句、小寫開頭、結尾不加句點，說**改了什麼**；標題（含 type）最多 72 字元。
- **不加 scope**：只有一個套件，寫 `fix:`，不寫 `fix(main):`。
- **一定要有 body**：主旨下空一行，用散文寫具體改了什麼（檔案、函式），再寫為什麼，以及什麼刻意沒動。
- 不相容的變更在 type 後面加 `!`（例如 `feat!:`），會列在 changelog 的「不相容變更」。
- 一筆 commit 只做一件事。

type 依**改動的性質**選，不是依改到哪種檔案：

| type | 用途 |
|---|---|
| `feat` | 使用者看得到的新功能或行為 |
| `fix` | 修正錯誤的行為 |
| `perf` | 只改善效能，行為不變 |
| `refactor` | 重構，行為和效能都不變 |
| `style` | **只有**不改變意義的格式調整（gofmt、golines、空白）。外觀或行為的改變是 `feat` 或 `fix`，即使只改了樣式檔 |
| `test` | 只動測試 |
| `docs` | 只動文件（README、CLAUDE.md、註解） |
| `build` | 建置系統或依賴（`go.mod`、`package.json`、`Taskfile.yaml`、`.goreleaser.yaml`） |
| `ci` | CI 設定（`.github/workflows/`） |
| `chore` | 其他不影響程式的雜務（lint、拼字字典、hook、編輯器設定） |
| `revert` | 還原先前的 commit |

GoReleaser 的 changelog 只列 `feat`、`fix`、`perf`、`revert`，以及標了 `!` 的 commit。

## 分支與發布

- 使用 git-flow-next，設定在可以 commit 的 `.gitflow`；修改後用 `git flow config sync` 同步到 `.git/config`。
- 嚴格的 GitHub Flow：只有 `main`（base）和 `feature/*`（topic）。不要加 `develop`、`release/*`、`hotfix/*`、`chore/*`、`fix/*`；修 bug、雜務也都用 `feature/*`。
- `feature.finish` 設定了 `keep = true`（finish 後保留分支）和 `push = true`（finish 後把 `main` 推到 `origin`）。保留分支是使用者的偏好，跟標準 GitHub Flow 不同，不要改回去。
- 發布就是在 `main` 上打 tag，沒有 release 分支。使用者的 git 設定了 `commit.gpgsign`，但沒有設定 `tag.gpgSign`，所以 `task release` 明確使用 `git tag --sign`。
- CI（`release.yaml`）的 lint 一定要設 `GOOS=windows`：程式只能在 Windows 編譯，在 Linux 上型別檢查會失敗，golangci-lint 卻仍然回報 0 issues，等於沒有檢查。

## 慣例

- 所有純文字檔都是 **LF、UTF-8、無 BOM**，`.gitattributes` 已固定 `eol=lf`。用 Python 寫檔時要注意 `newline` 和 `encoding`。
- 程式碼和註解用英文；README 以繁中為主。
- 除非使用者要求，否則不要 commit 或 push。
- `pnpm-workspace.yaml` 的 `allowBuilds` 允許 `@goreleaser/goreleaser` 執行 postinstall（用來下載執行檔）；pnpm 12 預設會擋。

## 驗證方式

沒有單元測試，因為行為完全取決於華碩注音。修改切換邏輯後要實測：

- **用實際打字驗證，不要只讀回狀態。** 在專用的測試視窗裡模擬打 `s` `u` `3` Enter：中文模式打出「你」，英文模式打出「su3」。同時對照 ini 和工作列指示器。
- **測試只能在自己開的視窗裡進行。** 用 WinForms 開一個有 TextBox 的視窗，用 `AttachThreadInput` 加 `SetForegroundWindow` 取得焦點（背景程式直接呼叫 `SetForegroundWindow` 會被前景鎖擋下）。
- **每個會送按鍵或執行切換的動作之前，都要檢查前景視窗還是不是測試視窗**，不是就中止。使用者常在測試途中切換視窗：曾經有 Shift 送進了使用者的 Discord，也有一次切換作用到了使用者正在用的視窗。
- 模擬 Shift 切換時，按下和放開之間要隔約 60 ms，太短會被華碩注音忽略。
- `asusimemodew.exe` 是 GUI 程式，從 PowerShell 用 `&` 執行時拿不到輸出和結束代碼，要用 `Start-Process -PassThru` 等它結束。
