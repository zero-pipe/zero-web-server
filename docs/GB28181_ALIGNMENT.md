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
  远程重启、强制关键帧、PTZ 精准控制、存储卡格式化、目标跟踪。
- `DeviceStatus` 查询：`GET /api/v1/device/query/devices/:deviceId/status`。
- `ConfigDownload` 查询：`QueryBasicParam` 读取设备实际应答，超时回退本地字段。

### 查询族（2022）
- 看守位 / 巡航轨迹列表 / 单条巡航轨迹 / 云台精准位置 / 存储卡状态：
  `GET /api/v1/device/query/{home_position|cruise_track_list|cruise_track|ptz_position|sdcard_status}`。
- 云台精准位置支持订阅与通知：`GET /api/v1/front-end/ptz_position/subscribe/:deviceId/:channelId`，
  推送值缓存在 `GET /api/v1/front-end/ptz_position/:deviceId/:channelId`。

### 级联（平台作为下级）
- 上级 `SUBSCRIBE`（Catalog/Alarm/MobilePosition）应答并回填 Expires；目录订阅后主动推送一次目录（默认 600s）。
- 上级 `Query`（Catalog/DeviceInfo/DeviceStatus）应答。
- 上级 `INVITE` 按 Offer 的 `s=` 分派实时/回放/下载；`INFO` 回放控制转发到叶子设备；`BYE` 收尾释放会话。
- **目录推送按组织树下发**：
  - 共享通道按"虚拟组织/业务分组 → 行政区划 → 通道"组装，父节点先于子节点；通道挂真实父节点。
  - 行政区划节点编号约定：`区划码右补0到8位 + 行业00 + 类型216 + 网络0 + 序号000000`（20 位）。
  - 快照差分，只发 `ADD/UPDATE/DEL`；祖先缺失时不发悬挂 ParentID，按根节点处理并记日志。
  - 按报文大小（约 700 字节）与 100 条上限分批，避免 UDP 超 MTU（附录 M/N）。
- 目录应答/通知按附录 J 补全 Manufacturer/Model/Parental/ParentID/RegisterWay/Secrecy/经纬度等字段。

### 级联（平台作为上级）
- 接收下级目录：入库 `zws_subordinate_catalog`，按 `(platform, deviceId)` 增量更新，`DEL` 删除。
- 父节点解析不到时标记 `orphan`、写告警日志、**不挂载**到树（数据保留，父节点补回后可自动挂上）。
- 只读接口：`GET /api/v1/subordinate/:id/catalog`（含 `query`/`orphan` 过滤）、`GET /api/v1/subordinate/:id/catalog/tree`。
- 已用两个实例对打验证：三级行政区划 + 虚拟组织 + 未挂点位成树、取消共享后 `DEL`、删除父节点后子节点转孤儿且只记日志。

### 图像抓拍（2022 9.14）
- `POST /api/v1/device/query/snap/config` 下发抓拍配置，自动生成 SessionID 与 UploadURL。
- `POST /api/v1/gb/snapshot/upload`（公开）接收设备 JPEG，按 41 位文件名解析设备并归档对象存储。
- `UploadSnapShotFinished` 通知记录，可查询 `GET /api/v1/device/query/snap/finished/:sessionId`。

### 其他
- 报警/MobilePosition/PTZPosition 通知接收；Catalog/Alarm/MobilePosition 订阅及自动续订。
- 报警解析 2022 扩展字段 `AlarmTypeParam` / `EventType` / `EventData`。
- TCP 注册连接断开立即判设备离线（NAT 内侧设备）。
- `OPTIONS` 返回 `Allow`，用于能力探测。

## 明确不做 / 暂缓

- **级联目录页面**：下级目录的入库、树接口已完成，但**前端页面未做**（当前用接口验证）。
- **公共安全式多级组织规划**：区域/业务分组/虚拟组织的**规划与共享授权策略**不预设，
  只按双方录入的规范国标码成树；非规范数据按"不挂 + 日志"处理，不做兼容映射。
- **语音广播与对讲**：按业务需要暂不实现（现有仅 Broadcast 通知 + WebRTC 推流，未做设备入站拉流）。
- **设备软件升级（9.13）**：按业务需要暂不实现。
- **报警分发到上级/终端**：需要先定分发路由策略（转发目标、预案），属产品决策，暂不实现。
- **编码增强（AAC/G.722.1/SVAC 的 SDP 声明）**：改动会触及已在海康设备上验证通过的实时点播 SDP，
  为避免回归暂缓；如后续需要非 PS 直连流再单独立项。
- **协议版本标识（2022 附录 I）**：缺少可机读的标准原文（PDF 字体嵌入导致文本无法提取），
  未按猜测实现；拿到附录 I 原文或样例报文后再做。

## 待验证

- 图像抓拍需支持 9.14 的设备验证上传与完成通知。
- 2022 查询族（看守位/巡航/云台位置/存储卡）需 2022 固件设备验证。
- 级联回放/下载与 INFO 控制（保留能力，暂不排期）。

## 与平台约定相关的说明
- 内码/外码/国标编号的展示规则见 [PLATFORM_CONVENTIONS.md](PLATFORM_CONVENTIONS.md)。
- 协议内部字段（国标编号、内码）仅用于收发与索引，不在页面暴露。
