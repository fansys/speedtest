// Package netguard 做 SSRF 防护：校验节点 address/port 是否允许被中心服务访问。
//
// 节点地址是「管理员输入」，中心服务会主动向它发起请求，属于典型 SSRF 面。这里做的事：
//
//  1. 只允许 http/https（由 ALLOWED_NODE_PROTOCOLS 控制），拒绝 file/gopher/ftp 等。
//  2. 拒绝带认证信息、路径、查询串的地址（address 只接受主机名或 IP 字面量）。
//  3. 端口必须在 1-65535，且不在明显危险的端口黑名单里。
//  4. 解析主机名得到全部 A/AAAA 记录，逐个检查：回环、私网、链路本地、多播、未指定
//     地址默认按 ALLOW_PRIVATE_NODES 开关处理（默认允许，因为内网自建测速节点是主要场景）。
//  5. 调用方在真正发请求前应当再次校验，缩小 DNS rebinding 窗口。
package netguard

import (
	"fmt"
	"net"
	"strings"
)

// BlockedPorts 即使在 ALLOW_PRIVATE_NODES=true 时也一律拒绝：
// 它们不是 HTTP 服务端口，指向它们通常意味着在拿本服务当跳板打内网组件。
var BlockedPorts = map[int]string{
	22:    "ssh",
	23:    "telnet",
	25:    "smtp",
	445:   "smb",
	587:   "smtp submission",
	3306:  "mysql",
	5432:  "postgres",
	6379:  "redis",
	9200:  "elasticsearch",
	11211: "memcached",
	27017: "mongodb",
}

// Error 表示节点地址不被允许。错误信息面向管理员，不含任何令牌。
type Error struct {
	msg string
}

func (e *Error) Error() string { return e.msg }

func newError(format string, args ...any) error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

// Resolved 是校验通过后归一化的结果。
type Resolved struct {
	Protocol  string
	Host      string
	Port      int
	Addresses []string
}

// BaseURL 返回可以直接用于发起请求的 base url。
func (r Resolved) BaseURL() string {
	host := r.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s://%s:%d", r.Protocol, host, r.Port)
}

func classify(ip net.IP) string {
	if ip.IsUnspecified() {
		return "未指定地址（0.0.0.0 / ::）"
	}
	if ip.IsLoopback() {
		return "回环地址"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "链路本地地址（含云元数据 169.254.169.254）"
	}
	if ip.IsMulticast() {
		return "多播地址"
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4.IsPrivate() {
			return "私网地址"
		}
	} else if ip.IsPrivate() {
		return "私网地址"
	}
	return ""
}

// ResolveAddresses 解析主机名到 IP 列表；host 本身是 IP 时直接返回。
func ResolveAddresses(host string) ([]string, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []string{host}, nil
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, newError("无法解析主机名 %q", host)
	}
	if len(ips) == 0 {
		return nil, newError("主机名 %q 没有解析到任何地址", host)
	}
	return ips, nil
}

// ValidateNodeTargetOptions 是 ValidateNodeTarget 的可选参数。
type ValidateNodeTargetOptions struct {
	AllowPrivate     bool
	AllowedProtocols []string
	Resolve          bool
}

// ValidateNodeTarget 校验节点地址，返回归一化后的结果。不通过则返回 *Error。
func ValidateNodeTarget(address string, port int, protocol string, opts ValidateNodeTargetOptions) (Resolved, error) {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	allowed := false
	for _, p := range opts.AllowedProtocols {
		if p == protocol {
			allowed = true
			break
		}
	}
	if !allowed {
		display := protocol
		if display == "" {
			display = "(空)"
		}
		return Resolved{}, newError("protocol 必须是 %s 之一，收到 %s", strings.Join(opts.AllowedProtocols, "/"), display)
	}

	host := strings.TrimSpace(address)
	if host == "" {
		return Resolved{}, newError("address 不能为空")
	}
	if strings.Contains(host, "://") {
		return Resolved{}, newError("address 只填主机名或 IP，不要带协议前缀")
	}
	for _, bad := range []string{"/", "?", "#", "@", "\\", " "} {
		if strings.Contains(host, bad) {
			return Resolved{}, newError("address 中不允许出现字符 %q", bad)
		}
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if strings.Contains(host, ":") {
		if net.ParseIP(host) == nil {
			return Resolved{}, newError("address 中不允许出现端口号或非法的 IPv6 字面量")
		}
	}
	if len(host) > 253 {
		return Resolved{}, newError("address 过长")
	}

	if port < 1 || port > 65535 {
		return Resolved{}, newError("port 必须在 1-65535 之间")
	}
	if _, blocked := BlockedPorts[port]; blocked {
		return Resolved{}, newError("端口 %d 在禁止列表中（非 HTTP 服务端口）", port)
	}

	if !opts.Resolve {
		return Resolved{Protocol: protocol, Host: host, Port: port}, nil
	}

	addresses, err := ResolveAddresses(host)
	if err != nil {
		return Resolved{}, err
	}
	if !opts.AllowPrivate {
		for _, addr := range addresses {
			ip := net.ParseIP(addr)
			if ip == nil {
				continue
			}
			if reason := classify(ip); reason != "" {
				return Resolved{}, newError("地址 %s 属于%s，当前 ALLOW_PRIVATE_NODES=false，已拒绝", addr, reason)
			}
		}
	}

	return Resolved{Protocol: protocol, Host: host, Port: port, Addresses: addresses}, nil
}
