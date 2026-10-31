# ASUS IME Mode

查詢或強制設定前景視窗中華碩注音的中／英模式。華碩注音只有「切換」中英的快速鍵，搭配這個程式就能做出「一按就是中文」、「一按就是英文」的快速鍵。

[English README](./README.en.md)

## 運作方式

大多數工具（例如 AutoHotkey 的 `IME_SetConvMode`、`im-select`）都是讀寫 IMM 的 conversion mode，但華碩注音完全不理會它。華碩注音的行為是：

- 每次切換中英，都會立刻把狀態寫進 `%ProgramFiles%\ASUS\AsusIME\zhtw\data\%UserName%\AsusIMEConfig.ini` 的 `[DirectOutputMode]`，以程式名稱為 key（`0` 是中文、`1` 是英文）。還沒有記錄的程式預設是中文。
- IMM 的 open status（`WM_IME_CONTROL` / `IMC_SETOPENSTATUS`）只要一改變，就會翻轉中英，不管寫入的值是什麼。

所以本程式從 ini 讀取前景程式目前的模式，只在跟要求的模式不同時翻轉一次 open status。過程中不模擬任何按鍵，觸發快速鍵時還按著的 Ctrl、Alt 等修飾鍵不會造成干擾。

所有中文輸入法的鍵盤配置語言都相同，從目標程式外部無法分辨正在使用哪一個。因此只要啟用了華碩注音以外的中文輸入法，本程式就會拒絕執行，以免誤判。

## 需求

- Windows 10 以上，且華碩注音是唯一啟用的中文輸入法
- 從快速鍵使用需要 Windows 11 24H2（build 26100）以上。舊版 Windows 從快速鍵啟動時會開出主控台視窗並搶走焦點，結果改到的是主控台本身（見「限制」）。在終端機和腳本中使用則沒有這個限制。

## 安裝

從 GitHub Releases 下載符合電腦架構的 zip，大多數電腦是 `windows_amd64`，ARM 筆電（例如 Snapdragon）是 `windows_arm64`。不確定的話，在 PowerShell 執行 `$env:PROCESSOR_ARCHITECTURE` 查看：`AMD64` 或 `ARM64`。

把 zip 解壓縮到 `%LOCALAPPDATA%\Programs\asus-ime-mode`，這是 Windows 個人安裝程式的慣用位置，不需要系統管理員權限。zip 裡只有一個程式 `asusimemode.exe`，終端機、腳本和快速鍵都用它：它的 manifest 設定了 [`consoleAllocationPolicy`](https://learn.microsoft.com/windows/console/console-allocation-policy)，從快速鍵啟動時不開主控台視窗，焦點留在要切換的視窗上；從終端機執行時則跟一般命令列程式一樣，會輸出結果和結束代碼。

要在終端機直接輸入 `asusimemode` 執行的話，把這個資料夾加進使用者的 PATH 環境變數。如果你已經有 `%UserProfile%\.local\bin` 並加進了 PATH（例如裝過 uv 或 Claude Code），也可以直接放在那裡。

更新時，把新版解壓縮到同一個資料夾並覆蓋舊檔。快速鍵設定記的是完整路徑，位置不變就不用重新設定。

從 v0.1.0 更新時：v0.1.0 另外附了給快速鍵用的 `asusimemodew.exe`，之後的版本已經拿掉了。請把快速鍵的應用程式改成 `asusimemode.exe`，再刪除舊的 `asusimemodew.exe`。

## 使用方式

```
asusimemode [zh|en|toggle]
```

- 不帶參數：輸出目前模式 `zh` 或 `en`。
- `zh`：強制切到中文。
- `en`：強制切到英文。
- `toggle`：切換中英。

切換後會輸出切換結果。

結束代碼：

| 代碼 | 意義 |
| ---- | ---- |
| 0 | 成功 |
| 2 | 參數錯誤 |
| 3 | 前景視窗沒有 IME 視窗 |
| 4 | 目前的鍵盤不是中文輸入法（例如美式鍵盤） |
| 5 | 500 ms 內模式沒有改變，通常是因為焦點所在的元件沒有輸入狀態（見「限制」） |
| 6 | 環境不支援：啟用了其他中文輸入法、沒有啟用華碩注音，或讀不到它的狀態 |

### 用 PowerToys 設定快速鍵

在 PowerToys 的「鍵盤管理員」選擇「重新對應快速鍵」，新增快速鍵並選擇「執行程式」動作：

- 應用程式：`%LOCALAPPDATA%\Programs\asus-ime-mode\asusimemode.exe`
- 引數：`zh` 或 `en`

## 限制

- 焦點所在的元件要有輸入狀態（input context）才能切換。輸入框都有；其他元件有沒有則看程式，有些程式即使焦點不在輸入框也能切換，有些程式的按鈕、網頁本文等則沒有，此時切換會以結束代碼 5 失敗。焦點移到沒有輸入狀態的元件時，華碩注音也可能自行切成英文，所以這時讀到的模式不代表回到輸入框後的模式。
- 模式是華碩注音依程式記錄的，同一個程式的多個視窗共用同一個模式。
- 對以系統管理員身分執行的視窗無效，除非本程式也以系統管理員身分執行。
- 依賴華碩注音未公開的行為，華碩注音更新後可能失效。
- Windows 11 24H2 以前的版本不認得 `consoleAllocationPolicy`，從快速鍵啟動時仍會開出主控台視窗，切換的就變成主控台而不是原本的視窗，所以不適合從快速鍵使用。
- 直接雙擊 `asusimemode.exe`，或在 PowerShell 用 `Start-Process` 卻沒加 `-NoNewWindow` 執行時，同樣不會開視窗，因此看不到輸出。

## 開發

需要：

- Go 1.27 以上
- [Task](https://taskfile.dev)：所有指令都透過 `task` 執行，`task --list` 列出全部
- [golangci-lint](https://golangci-lint.run) v2：`task fmt` 格式化、`task lint` 檢查
- Node.js 與 pnpm：先執行一次 `pnpm install`，安裝 cspell、[husky](https://typicode.github.io/husky/)、commitlint 與 GoReleaser。pre-commit hook 會檢查拼字，暫存了 Go 相關檔案時再執行 `golangci-lint run`；commit-msg hook 會用 commitlint 檢查 commit message 是否符合 [Conventional Commits](https://www.conventionalcommits.org/zh-hant/)（type 採 Angular convention，規則見 `.commitlintrc.yaml`）

`task build` 會把程式編譯到 repo 裡的 `bin/`（已列在 `.gitignore`），不會安裝到任何地方；要放到自己電腦上執行的位置，可以寫一支 `*.local.ps1` 腳本（同樣被 git 忽略）來複製。原始碼沒有變更時會略過編譯，加上 `--force` 強制重新編譯。

manifest 在 `winres/asusimemode.manifest`。Go 的連結器不能直接嵌入 manifest，所以它被編譯成 `rsrc_windows_amd64.syso`、`rsrc_windows_arm64.syso` 並 commit 進來，`go build`、`go install` 和 GoReleaser 會自動連結，編譯時不需要其他工具。修改 manifest 後要執行 `task winres` 重新產生 `.syso`，這一步需要 Visual Studio Build Tools 的 C++ 工作負載（用其中的 `rc.exe` 和 `cvtres.exe`）；`task winres:check` 檢查 commit 的 `.syso` 是否跟 manifest 一致，pre-commit hook 在暫存了 `winres/` 或 `.syso` 時會執行它。

### 發布

分支採用 GitHub Flow（設定在 `.gitflow`，給 [git-flow-next](https://git-flow.sh) 使用）：只有 `main` 和 `feature/*`，沒有 release 分支，發布就是在 `main` 上打 tag。

1. 在 `main` 上執行 `task release VERSION=0.1.0`（版本號照 [SemVer](https://semver.org/lang/zh-TW/)，不加 `v`）。它會先確認：
   - 目前在 `main`、工作目錄沒有未提交的變更、跟 `origin/main` 一致
   - `v0.1.0` 這個 tag 在本機和 `origin` 都還不存在

   接著執行 lint、拼字檢查和 `task release:snapshot`，全部通過後建立有簽章的 `v0.1.0` tag 並推送。
2. tag 推送後，GitHub Actions 會先在 CI 上再檢查一次：tag 是否在 `main` 上、lint、拼字。通過後才用 GoReleaser 編譯並建立 GitHub Release，amd64 和 arm64 各一個 zip，另附 `checksums.txt`。

只想確認發布內容、不發布的話，執行 `task release:snapshot`，結果會放在 `dist/`。

## 授權

[MIT](./LICENSE)
