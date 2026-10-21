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

## 安裝

從 GitHub Releases 下載符合電腦架構的 zip，大多數電腦是 `windows_amd64`，ARM 筆電（例如 Snapdragon）是 `windows_arm64`。不確定的話，在 PowerShell 執行 `$env:PROCESSOR_ARCHITECTURE` 查看：`AMD64` 或 `ARM64`。

把 zip 解壓縮到 `%LOCALAPPDATA%\Programs\asus-ime-mode`，這是 Windows 個人安裝程式的慣用位置，不需要系統管理員權限。zip 裡有兩個程式：

- `asusimemode.exe`：主控台版，在終端機和腳本中使用。
- `asusimemodew.exe`：GUI 版，給快速鍵使用。主控台版從快速鍵啟動時會開出主控台視窗，搶走目標視窗的焦點，結果改到的是主控台本身。

要在終端機直接輸入 `asusimemode` 執行的話，把這個資料夾加進使用者的 PATH 環境變數。如果你已經有 `%UserProfile%\.local\bin` 並加進了 PATH（例如裝過 uv 或 Claude Code），也可以直接放在那裡。

更新時，把新版解壓縮到同一個資料夾並覆蓋舊檔。快速鍵設定記的是完整路徑，位置不變就不用重新設定。

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
| 5 | 500 ms 內模式沒有改變 |
| 6 | 環境不支援：啟用了其他中文輸入法、沒有啟用華碩注音，或讀不到它的狀態 |

### 用 PowerToys 設定快速鍵

在 PowerToys 的「鍵盤管理員」選擇「重新對應快速鍵」，新增快速鍵並選擇「執行程式」動作：

- 應用程式：`%LOCALAPPDATA%\Programs\asus-ime-mode\asusimemodew.exe`
- 引數：`zh` 或 `en`

## 限制

- 模式是華碩注音依程式記錄的，同一個程式的多個視窗共用同一個模式。
- 對以系統管理員身分執行的視窗無效，除非本程式也以系統管理員身分執行。
- 依賴華碩注音未公開的行為，華碩注音更新後可能失效。

## 開發

需要：

- Go 1.27 以上
- [Task](https://taskfile.dev)：所有指令都透過 `task` 執行，`task --list` 列出全部
- [golangci-lint](https://golangci-lint.run) v2：`task fmt` 格式化、`task lint` 檢查
- Node.js 與 pnpm：先執行一次 `pnpm install`，安裝 cspell、[husky](https://typicode.github.io/husky/)、commitlint 與 GoReleaser。pre-commit hook 會檢查拼字，暫存了 Go 相關檔案時再執行 `golangci-lint run`；commit-msg hook 會用 commitlint 檢查 commit message 是否符合 [Conventional Commits](https://www.conventionalcommits.org/zh-hant/)（type 採 Angular convention，規則見 `.commitlintrc.yaml`）

`task build` 會把兩個程式編譯到 `%UserProfile%\.local\bin`，可用 `task build BIN_DIR=<資料夾>` 改變位置；`task build:console`、`task build:gui` 則只編譯其中一個。原始碼沒有變更時會略過編譯，加上 `--force` 強制重新編譯。

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
