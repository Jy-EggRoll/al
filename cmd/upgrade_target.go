package cmd

import "github.com/jy-eggroll/eggokit/upgradecli"

// alTarget 声明 al 自升级的发布目标，交给 eggokit/upgradecli 的共享升级命令使用。
//
// 这些取值属于 al 自己的发布约定，与升级器的通用能力无关，所以留在命令层。
// 平台表必须与仓库根的 release.json 保持同步：清单里多一个平台，构建会产出一份
// 升级器不认识的资产；这里多一个平台，用户会拿到一个必然下载失败的地址。
// 两个方向的偏差都在 cmd/release_manifest_test.go 里被逐项比对，前提是两边同时改。
//
// AssetPrefix 必须与构建产物名 al-<os>-<arch>[.exe] 一致：改名产物或改名这里的任一处，
// 升级器就匹配不到资产，并把「找不到资产」报成「已是最新」。
var alTarget = upgradecli.Target{
	Owner: "Jy-EggRoll",
	Repo:  "al",

	AssetPrefix: "al",

	Platforms: map[string]map[string]bool{
		"windows": {"386": true, "amd64": true, "arm64": true},
		"linux":   {"386": true, "amd64": true, "arm": true, "arm64": true},
		"darwin":  {"amd64": true, "arm64": true},
		"freebsd": {"amd64": true, "arm64": true},
	},

	// 直连 GitHub 不稳定时供用户选择的备用下载代理前缀，与资产原始地址直接拼接。
	// 是否启用完全由用户在升级过程中的确认决定，程序不会静默换源：
	// 下载结果随后会被直接替换成本机的可执行文件，用户必须清楚二进制来自哪里
	ProxyPrefix: "https://gh-proxy.org/",

	// 请求标识用程序名，便于上游区分流量来源
	UserAgent: "al-updater",
}
