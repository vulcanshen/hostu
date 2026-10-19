# hostu — build / run / test / package（家族共用的 target 在 family.mk）
#
# 跑 `make`（或 `make help`）列出所有指令。

BINARY := hostu
CHECK  := cross
include family.mk

##@ hostu 自己的

.PHONY: cross
cross: ## 照 Windows 與 Linux 再 vet 一次（hostu 出 Windows 版；在 macOS 上抓只在別的平台才編得到的錯）
	GOOS=windows go vet ./...
	GOOS=linux go vet ./...

.PHONY: sandbox
sandbox: build ## 在暫存目錄裡跑 hostu：hosts 檔是一份示範檔，設定、資料、家目錄都是新的
	tools/sandbox.sh ./$(BINARY)
