package cmd

import (
	"testing"

	"github.com/jy-eggroll/eggokit/release"
)

// TestReleaseManifestMatchesSupportedPlatforms 固定「发布清单」与「升级器已知平台」这两份定义完全一致。
//
// 回归背景：「有哪些平台有产物」这一事实同时存在于两处——仓库根的 release.json（驱动构建，
// 由 eggokit 的 buildall 读取）与 cmd/upgrade_target.go 的 alTarget.Platforms（驱动升级器挑资产）。
// 二者一旦分叉就会制造用户可见的故障：清单多一个平台，构建会产出一份升级器不认识的资产；
// alTarget.Platforms 多一个平台，升级器会给用户一个必然下载失败的地址。
//
// 断言方式：双向子集检查（清单 ⊆ 已知平台，且已知平台 ⊆ 清单），两个方向都成立即集合相等。
// 反向检查不能省：只验清单 ⊆ 已知平台的话，升级器这边多加一个平台不会被发现。
//
// cmd 下的测例工作目录是 cmd/，因此清单路径写成 ../release.json。
func TestReleaseManifestMatchesSupportedPlatforms(t *testing.T) {
	m, err := release.Load("../release.json")
	if err != nil {
		t.Fatalf("加载 ../release.json 失败: %v", err)
	}

	// 方向一：清单里声明的平台必须都被 alTarget.Platforms 认识，
	// 否则构建会产出升级器认不出的资产，用户升级时被告知「尚未提供该平台的产物」
	for _, p := range m.Platforms {
		if !alTarget.Platforms[p.OS][p.Arch] {
			t.Errorf("清单声明了平台 %s，但 alTarget.Platforms 不认识它：构建会产出升级器无法识别的资产", p.String())
		}
	}

	// 方向二：alTarget.Platforms 里的每个组合都必须在清单里出现，
	// 否则升级器会把一个不存在的下载目标交给用户
	for goos, arches := range alTarget.Platforms {
		for goarch := range arches {
			if !m.Supports(goos, goarch) {
				t.Errorf("alTarget.Platforms 声明了 %s/%s，但清单里没有它：升级器会给出不存在的下载目标", goos, goarch)
			}
		}
	}

	// 资产名一致性：清单推导出的产物名必须与 alTarget.AssetName 逐字相同（含 windows 的 .exe）。
	// 二者分叉的后果最难排查——「发布了但升级器认不出资产」，因此这条单独固定
	for _, p := range m.Platforms {
		want := m.AssetName(p)
		got, ok := alTarget.AssetName(p.OS, p.Arch)
		if !ok {
			t.Errorf("alTarget.AssetName(%s) 报不支持，但清单声明了该平台", p.String())
			continue
		}
		if got != want {
			t.Errorf("资产名不一致：alTarget.AssetName(%s)=%q，manifest.AssetName=%q", p.String(), got, want)
		}
	}
}
