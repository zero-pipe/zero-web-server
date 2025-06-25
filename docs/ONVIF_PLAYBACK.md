# ONVIF 播放流程与稳定性说明

## 1. 与 GB28181 的区别

| 项目 | GB28181 | ONVIF（本系统） |
|------|---------|-----------------|
| 信令 | SIP INVITE | ONVIF GetStreamURI |
| 媒体 | 摄像机 **推** RTP/PS 到 ZMS | ZMS **拉** 摄像机 RTSP |
| ZMS 流名 | `rtp/{deviceId}_{channelId}` | `onvif/{deviceId}_{profileToken}` |
| 前端播放器 | Jessibuca / WebRTC / H265web | 同上（按编码自动选择） |

## 2. 端到端流程

```
用户点击「播放」
  → 前端 GET /api/onvif/play/start?channelId=
  → Go ONVIF Service
       1) 若 ZMS 已有 onvif/{id}_{profile} 在线流 → 直接返回播放地址（复用）
       2) 否则 GetStreamURI 得到 RTSP
       3) ZMS addStreamProxy（RTSP over TCP, auto_close=关）
       4) 等待 ZMS 出流，读取真实 videoCodec（H264/H265）
  → 返回 { app, stream, videoCodec, urls: { flv, ws, ... } }
  → 前端 playerTabs（`web/src/utils/playerStrategy.js`）
       H264 → Jessibuca（HTTP-FLV 优先）
       H265 → H265web
       WebRTC 就绪后：Opus 等音频或无声流优先 WebRTC（当前未启用）
  → 浏览器连接 http://{media_ip}:8080/onvif/{stream}.flv
```

**停止：**

```
用户点击「停止」或关闭弹窗
  → 前端 destroy 播放器
  → GET /api/onvif/play/stop → ZMS CloseStreams
  → 断开 RTSP 拉流，释放摄像机连接
```

## 3. 海康 Profile 与 RTSP 通道（重要）

海康摄像机常见 RTSP 路径：

| RTSP 通道 | 含义 | 典型编码 |
|-----------|------|----------|
| `/Streaming/Channels/101` | **主码流** | H264 高清 |
| `/Streaming/Channels/102` | **子码流** | H264 或 H265（以摄像机配置为准） |

ONVIF Profile 名称（`mainStream` / `subStream`、`Profile_1` / `Profile_2`）**不等于** 101/102，二者可能交叉映射。

**本系统规则：**

- 列表「码流」按 RTSP 路径识别：**101=主码流，102=子码流**
- 「配置编码 / 配置分辨率」来自 ONVIF Profile（可能滞后或不准确）
- **播放时以 ZMS 实测 `videoCodec` 为准**（解析 RTSP RTP 载荷，与日志 `first RTP codec=H265/H264` 一致）

若配置显示 H264 但 ZMS 实测 H265，请在摄像机 Web **视频编码** 里确认对应码流，或直接使用 H265web。

## 4. 播放器选择

| 编码 | 推荐播放器 | 说明 |
|------|------------|------|
| H264 | Jessibuca | 首帧快，适合子码流/主码流 H264 |
| H265 | H265web | 需加载 WASM，首帧较慢；**不要用 Jessibuca** |

ONVIF 同步通道时 `codec` 字段来自 Profile 配置，可能与 RTSP 实际编码不一致（例如表内显示 H264，ZMS 实际拉到 H265）。  
**以 `/api/onvif/play/start` 返回的 `videoCodec`（ZMS 实测）为准**；`configCodec` 为 ONVIF 配置值供对照。

## 5. 稳定性机制（已实现）

### 4.1 切换播放器不断流

切换 Jessibuca ↔ H265web 时，旧播放器会先断开 FLV，ZMS 会短暂出现 `none_reader`。

- **国标 `rtp` 与 ONVIF `onvif`**：hook `on_stream_none_reader` 返回 `close=false`，不因无人观看拆掉流。
- 避免「第一次能播、一切换就黑屏、再切 Jessibuca 不能播」。

### 4.2 重复播放复用代理

同一通道再次点「播放」时，若 ZMS 上流仍在，**跳过 addStreamProxy**，避免 `kick_prev` 重拉 RTSP。

### 4.3 RTSP 拉流参数

- `rtp_type=1`：RTSP over TCP（海康等仅 TCP 场景）
- `auto_close=false`：不在 ZMS 侧因无人观看自动删除 pull proxy

### 4.4 前端生命周期

- 关闭弹窗 / 切换通道：`playerTabs.destroy()`，再调 stop API
- 切换播放器 tab：先 `pause` 旧实例再启新实例

## 5. 日志对照（ZMS）

| 日志 | 含义 |
|------|------|
| `live_pull_proxy register onvif/1_Profile_2` | 注册 RTSP 拉流代理 |
| `video=H265` / `video=H264` | **实际编码**，决定播放器 |
| `http-flv live play ... readers=1` | 浏览器开始播 FLV |
| `none_reader app=onvif` | 暂无观众（切换播放器时正常） |
| `source_clear app=onvif` | 流被清掉（点停止 / CloseStreams / 异常关流） |
| `publish_reregister ... kick_prev` | 同一 stream 重复 addStreamProxy（应尽量避免，已做复用） |

## 6. 常见问题

**Q: 第一次 Jessibuca 很快，H265web 慢？**  
A: H265web 要拉 WASM，首帧 2～5 秒正常；H265 码流应固定用 H265web。

**Q: 第二次 Jessibuca 不能播？**  
A: 常见原因：① 码流实为 H265 却用 Jessibuca；② 切换播放器时流被 none_reader 关掉（已修复）；③ 未 destroy 旧播放器导致多路 FLV 连接抖动。

**Q: subStream / mainStream 表内编码不准？**  
A: 以 play/start 返回的 `videoCodec` 为准；必要时在摄像机 Web 确认 Profile 编码。

**Q: 与 `stream_on_demand` 的关系？**  
A: 国标点播受 `configs/config.yaml` 中 `stream_on_demand` 影响；ONVIF 拉流在 none_reader hook 中**强制不关流**，仅用户点「停止」或关弹窗时 CloseStreams。

## 7. 相关代码

- 后端播放：`internal/application/onvif/service.go` → `StartPlay` / `StopPlay`
- ZMS 代理：`internal/infrastructure/media/mediakit/client.go` → `AddStreamProxy`
- 无人观看 hook：`internal/interfaces/hook/zlm_handler.go` → `onStreamNoneReader`
- 前端：`web/src/views/onvifDevice/channel/index.vue`
