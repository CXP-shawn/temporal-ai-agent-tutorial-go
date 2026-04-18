// codec/claim_check.go
// -----------------------------------------------------------------------------
// 【这是什么】一个 Temporal PayloadCodec:专门处理"超大 payload"。
// 【类比】**行李寄存牌**:
//   机场带 20kg 行李上飞机,比把行李拖进驾驶舱要聪明 ——
//   把箱子放寄存处,登机牌只写一个"取件码"。到目的地凭码取行李就行。
//
//   Temporal 的 Workflow 历史也一样:
//     - 默认把 Activity 的输入输出原样塞进历史
//     - 一旦 payload 很大(>128KB),历史迅速膨胀,重放会变慢、存储贵
//     - 我们拦截一下:payload 超阈值时,写到磁盘 / S3 / OSS,只在历史里留一个"取件码"
//
// 【在哪层】基础设施层。在 main.go 创建 client 时注入 DataConverter 即可。
// 【注意】这个示例为了最小化依赖,用本地临时目录做存储。生产应换成对象存储。
//         另外 sha256 只用于生成文件名,不做安全校验。
// -----------------------------------------------------------------------------

package codec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// 超过 128KB 就外置
const thresholdBytes = 128 * 1024

// metadata key,用来标记"这条 payload 其实是取件码"
const claimCheckMetaKey = "claim-check-path"

// ClaimCheckCodec 实现 converter.PayloadCodec。
type ClaimCheckCodec struct {
	// Dir 是寄存处目录。默认为 os.TempDir()/claim-check。
	Dir string
}

// NewClaimCheckCodec 构造一个编码器,自动创建目录。
func NewClaimCheckCodec() (*ClaimCheckCodec, error) {
	dir := filepath.Join(os.TempDir(), "claim-check")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &ClaimCheckCodec{Dir: dir}, nil
}

// Encode 发送前被调用:大 payload → 寄存 + 替换为取件码。
func (c *ClaimCheckCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	out := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		if len(p.GetData()) < thresholdBytes {
			out[i] = p // 小的原样透传
			continue
		}
		// 用 sha256 算出文件名(行李牌号)
		sum := sha256.Sum256(p.GetData())
		name := hex.EncodeToString(sum[:]) + ".bin"
		path := filepath.Join(c.Dir, name)

		// 真正把行李放到寄存处
		if err := os.WriteFile(path, p.GetData(), 0o644); err != nil {
			return nil, fmt.Errorf("写入 claim-check 文件失败: %w", err)
		}

		// 新建一个极小的替身 payload:
		//   - metadata 带上 "claim-check-path = <绝对路径>"
		//   - data 留空(或保留 encoding 信息,这里简化)
		newPayload := &commonpb.Payload{
			Metadata: map[string][]byte{
				claimCheckMetaKey: []byte(path),
			},
			Data: nil,
		}
		// 原始 encoding 也带上,方便 Decode 还原
		for k, v := range p.Metadata {
			if _, exists := newPayload.Metadata[k]; !exists {
				newPayload.Metadata[k] = v
			}
		}
		out[i] = newPayload
	}
	return out, nil
}

// Decode 接收前被调用:遇到取件码 → 从寄存处读回原始行李。
func (c *ClaimCheckCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	out := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		pathBytes, ok := p.Metadata[claimCheckMetaKey]
		if !ok {
			out[i] = p // 没有取件码,直接放行
			continue
		}
		data, err := os.ReadFile(string(pathBytes))
		if err != nil {
			return nil, fmt.Errorf("读取 claim-check 文件失败: %w", err)
		}
		// 复原:去掉 claim-check 标记,把 data 塞回来
		meta := make(map[string][]byte, len(p.Metadata)-1)
		for k, v := range p.Metadata {
			if k == claimCheckMetaKey {
				continue
			}
			meta[k] = v
		}
		out[i] = &commonpb.Payload{Metadata: meta, Data: data}
	}
	return out, nil
}

// 编译期断言:ClaimCheckCodec 必须满足 converter.PayloadCodec 接口
var _ converter.PayloadCodec = (*ClaimCheckCodec)(nil)
