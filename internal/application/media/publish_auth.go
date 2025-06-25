package media

import (
	"net/url"
	"strings"

	"zero-web-kit/internal/infrastructure/persistence"
)

const (
	RTPApp      = "rtp"
	LoadMP4App  = "mp4_record"
	TalkApp     = "talk"
	BroadcastApp = "broadcast"
)

type PublishAuth struct {
	userRepo       *persistence.UserRepository
	pushRepo       *persistence.StreamPushRepository
	proxyRepo      *persistence.StreamProxyRepository
	pushAuthority  bool
	recordPushLive bool
	publish        *PublishRegistry
}

func NewPublishAuth(
	userRepo *persistence.UserRepository,
	pushRepo *persistence.StreamPushRepository,
	proxyRepo *persistence.StreamProxyRepository,
	pushAuthority, recordPushLive bool,
	publish *PublishRegistry,
) *PublishAuth {
	return &PublishAuth{
		userRepo: userRepo, pushRepo: pushRepo, proxyRepo: proxyRepo,
		pushAuthority: pushAuthority, recordPushLive: recordPushLive,
		publish: publish,
	}
}

type PublishResult struct {
	Allowed    bool
	EnableMP4  bool
	EnableAudio bool
}

func (a *PublishAuth) Authenticate(app, stream, params string) PublishResult {
	res := PublishResult{Allowed: true, EnableAudio: true}
	if app == RTPApp {
		if a.publish != nil && a.publish.ShouldMP4(app, stream) {
			res.EnableMP4 = true
		}
		return res
	}
	if app == LoadMP4App || app == TalkApp || app == BroadcastApp {
		return res
	}
	if proxy, err := a.proxyRepo.GetByAppStream(app, stream); err == nil {
		res.EnableMP4 = proxy.EnableMP4
		res.EnableAudio = proxy.EnableAudio
		return res
	}
	if _, err := a.pushRepo.GetByAppStream(app, stream); err == nil {
		if a.recordPushLive {
			res.EnableMP4 = true
		}
		return res
	}
	if a.pushAuthority {
		callID, sign := parseParams(params)
		if sign == "" || !a.userRepo.CheckPushAuthority(callID, sign) {
			return PublishResult{Allowed: false}
		}
	}
	if a.recordPushLive {
		res.EnableMP4 = true
	}
	return res
}

func parseParams(raw string) (callID, sign string) {
	if raw == "" {
		return "", ""
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		if strings.Contains(raw, "sign=") {
			for _, part := range strings.Split(raw, "&") {
				if strings.HasPrefix(part, "sign=") {
					sign = strings.TrimPrefix(part, "sign=")
				}
				if strings.HasPrefix(part, "callId=") {
					callID = strings.TrimPrefix(part, "callId=")
				}
			}
		}
		return callID, sign
	}
	return vals.Get("callId"), vals.Get("sign")
}
