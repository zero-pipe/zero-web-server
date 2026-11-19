# GB/T 28181（2016 / 2022）对齐情况

本文记录平台当前与 GB/T 28181-2016、GB/T 28181-2022 的对齐程度，作为后续补齐的依据。
协议实现集中在子模块 `3rdpart/gb28181-go`，平台侧适配在 `internal/infrastructure/sip`、
`internal/application/*` 与 `internal/port`。

## 已对齐

### 注册 / 注销 / 鉴权（2016 9.1）
- 摘要挑战-应答；服务器签发的 nonce 参与校验，报文自带 nonce 不匹配时回 `401 stale=true`，拒绝重放。
- 注销（`Expires:0`）与注册走同一套挑战-应答，避免伪造报文踢设备下线。
- `Expires` 缺省 86400、下限 3600（`Config.DefaultExpires` / `MinExpires` 可配）。
- SIP 校时 `Date` 头使用北京时间 `yyyy-MM-ddTHH:mm:ss.SSS`（9.10）。
- 向上级注册：按有效期 2/3 刷新；失败间隔 60s 重试；心跳连续失败后重新注册（9.1.1）。
- 注册重定向 302（2022 9.1.2.3）：服务端可通过 `Config.RedirectPolicy` 下发 `302 Moved Temporarily`
  + `Contact` + `Expires`；级联客户端可自动跟随 `Contact` 重注册（最多 3 跳）。
  平台侧通过 `sip.redirect_contact` 配置启用。

### 信令字符集（2016 6.10 / 2022）
- 入站按 XML 声明把 GB2312 / GBK / GB18030 归一化为 UTF-8；出站 MANSCDP 报文编码为 GB18030 并同步改写声明。

### 目录（9.5 / 附录 J·N）
- 目录查询应答全量重建通道。
- 目录 `NOTIFY` 按 `Item.Event` 增量处理：`ADD/UPDATE` upsert、`DEL` 删除单条、`ON/OFF/VLOST/DEFECT` 仅更新状态，
  不再整表重建导致误删其它通道。用户自定义通道名在增量与全量同步中均保留。

### 实时点播 / 回放 / 下载
- SDP 含 `f=` 媒体描述与 H.265 PT100 声明。
- 历史回放/下载 SSRC 首位为 1，实时点播首位为 0。
- `MediaStatus`（NotifyType=121，File to End）触发会话自动收尾与媒体释放。
- 支持 TCP 被动/主动传输与 MANSRTSP 播放控制（PLAY/PAUSE/Scale/Seek）。

### 设备控制
- 云台 PTZ、预置位、前端控制、录像控制、布防/撤防、拉框缩放、看守位、基础参数配置、报警复位、
  远程重启、强制关键帧。
- `DeviceStatus` 查询已接线：`GET /api/v1/device/query/devices/:deviceId/status`。

### 其他
- 报警/MobilePosition 通知接收；Catalog/Alarm/MobilePosition 订阅及自动续订。
- `OPTIONS` 返回 `Allow`，用于能力探测。

## 待补齐（按优先级）

### P1（2016 已要求）
- `ConfigDownload` 应答解析（当前 `QueryBasicParam` 只发送查询，返回本地库字段）。
- 报警业务分发：按目标路由把报警转发给上级/终端，并回填 `Alarm` 业务应答（图 15）。
- 级联方向“被订阅/被查询”：入向 `SUBSCRIBE` 与入向 `Query(Catalog/DeviceInfo/DeviceStatus)` 应答。
- 级联入站回放/下载：解析 Offer 的 `s=Playback/Download`、`t=/u=` 并回填对应 `s=`。
- 语音广播/对讲：真实设备入站 `INVITE` 拉流（当前仅通知 + WebRTC 推流），目录上报 137 语音输出设备。
- TCP 长连接断线判定（NAT 内侧设备）。

### P2（2022 新增 / 增强）
- 2022 查询族接线：HomePositionQuery / CruiseTrackList / CruiseTrack / PTZPosition（构造器已有，缺解析与路由）、
  SDCardStatus。
- PTZ 精准控制、存储卡格式化、目标跟踪；PTZ 精准位置事件订阅与通知。
- 设备软件升级（9.13）：升级命令、文件获取、结果通知。
- 图像抓拍（9.14）：抓拍配置下发、`UploadSnapShotFinished`、41 位文件名规则。
  注意：现有 `snap` 是平台侧截帧存档，与此不是一回事。
- 协议版本标识（2022 附录 I）。
- 编码增强：AAC / G.722.1 / SVAC 的 SDP 声明与目录 EncodeType。
- 报警 `AlarmTypeParam` 扩展字段解析、`MediaStatus` 之外的设备状态变化聚合。

## 与平台约定相关的说明
- 内码/外码/国标编号的展示规则见 [PLATFORM_CONVENTIONS.md](PLATFORM_CONVENTIONS.md)。
- 协议内部字段（国标编号、内码）仅用于收发与索引，不在页面暴露。
