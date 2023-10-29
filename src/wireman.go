package main

import (
	"log"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"time"
	"net/http"
	"io"
)

type Wireman struct {
	InterfaceName string
	Ipt           *iptables.IPTables
	ListenPort    int
	Config        *WireguardConfig
}

func MakeWireman(interfaceName string, listenPort int) *Wireman {
	wireman := &Wireman{
		InterfaceName: interfaceName,
		ListenPort:    listenPort,
	}

	if !config.DisableKillswitch {
		wireman.SetupIptables()
	}

	return wireman
}

func (wm *Wireman) Up(profile *WireguardConfig) {
	log.Println("Bringing up wireguard interface:", wm.InterfaceName)
	wm.Config = profile

	if !config.DisableKillswitch {
		wm.Allow(profile.EndpointAddress)
	}

	wm.CreateDevice()
	wm.ConfigureDevice()
	wm.ConfigureRoutes()
}

func (wm *Wireman) CreateDevice() error {
	tdev, err := tun.CreateTUN(wm.InterfaceName, 1420)

	if err != nil {
		log.Panic(err)
	}

	device := device.NewDevice(tdev, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
	fileUAPI, err := ipc.UAPIOpen(wm.InterfaceName)

	if err != nil {
		log.Panic(err)
	}

	errs := make(chan error)

	log.Println("Starting UAPI listener")
	uapi, err := ipc.UAPIListen(wm.InterfaceName, fileUAPI)

	if err != nil {
		log.Panic("Failed to listen on uapi socket:", err)
	}

	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				errs <- err
				return
			}
			go device.IpcHandle(conn)
		}
	}()

	return nil
}

func (wm *Wireman) SetupIptables() {
	log.Println("Setting up iptables")
	ipt, err := iptables.New()

	if err != nil {
		log.Panic(err)

	}

	if err := ipt.ClearChain("filter", "OUTPUT"); err != nil {
		log.Panic(err)
	}

	if err := ipt.ChangePolicy("filter", "OUTPUT", "DROP"); err != nil {
		log.Panic(err)
	}

	if err := ipt.AppendUnique("filter", "OUTPUT", "-o", "wg0", "-j", "ACCEPT"); err != nil {
		log.Panic(err)
	}

	if err := ipt.AppendUnique("filter", "OUTPUT", "-o", "lo", "-j", "ACCEPT"); err != nil {
		log.Panic(err)
	}

	wm.Ipt = ipt
}

func (wm *Wireman) Allow(ip string) {
	log.Println("Allowing IP:", ip)

	if err := wm.Ipt.AppendUnique("filter", "OUTPUT", "-d", ip, "-j", "ACCEPT"); err != nil {
		log.Panic(err)
	}
}

func (wm *Wireman) ToggleDNS(enabled bool) {
	log.Println("Toggling DNS to:", enabled)
	rulespec := []string{"-p", "udp", "-m", "udp", "--dport", "53", "-j", "ACCEPT"}

	if enabled {
		err := wm.Ipt.AppendUnique("filter", "OUTPUT", rulespec...)
		FatalError(err)
	} else {
		err := wm.Ipt.Delete("filter", "OUTPUT", rulespec...)
		FatalError(err)
	}
}

func (wm *Wireman) ConfigureDevice() {
	log.Println("Configuring device")
	privateKey, err := wgtypes.ParseKey(wm.Config.PrivateKey)
	FatalError(err)

	publicKey, _ := wgtypes.ParseKey(wm.Config.PublicKey)
	addr, err := netip.ParseAddrPort(wm.Config.EndpointAddress + ":" + strconv.Itoa(wm.Config.EndpointPort))
	FatalError(err)

	udpAddr := net.UDPAddrFromAddrPort(addr)
	_, ipnet, err := net.ParseCIDR(wm.Config.AllowedIPs[0])
	FatalError(err)

	config := wgtypes.Config{
		PrivateKey:   &privateKey,
		ReplacePeers: true,
		ListenPort:   &wm.ListenPort,
		Peers: []wgtypes.PeerConfig{wgtypes.PeerConfig{
			PublicKey:  publicKey,
			Endpoint:   udpAddr,
			AllowedIPs: []net.IPNet{*ipnet},
		}},
	}

	client, err := wgctrl.New()
	FatalError(err)
	err = client.ConfigureDevice(wm.InterfaceName, config)
	FatalError(err)
}

func (wm *Wireman) ConfigureRoutes() {
	log.Println("Configuring routes")
	link, err := netlink.LinkByName(wm.InterfaceName)
	FatalError(err)

	mainLink, _ := netlink.LinkByName("eth0")
	addr, err := netlink.ParseAddr(wm.Config.Address)
	FatalError(err)

	netlink.AddrAdd(link, addr)
	netlink.LinkSetUp(link)

	_, defaultDst, _ := net.ParseCIDR("0.0.0.0/1")
	route := netlink.Route{Dst: defaultDst, LinkIndex: link.Attrs().Index}

	if err := netlink.RouteAdd(&route); err != nil {
		log.Panic(err)
	}

	_, defaultDst, _ = net.ParseCIDR(wm.Config.Address)
	defaultGateway := GetDefaultGateway()

	route = netlink.Route{
		Dst:       defaultDst,
		LinkIndex: mainLink.Attrs().Index,
		Gw:        net.ParseIP(defaultGateway),
	}

	if err := netlink.RouteAdd(&route); err != nil {
		log.Println("route already exists", defaultGateway, "-", wm.Config.Address)
	}
}

func GetDefaultGateway() string {
	iface, err := net.InterfaceByName("eth0")

	if err != nil {
		log.Panic(err)
	}

	addrs, _ := iface.Addrs()
	cidr := addrs[0].String()
	ip, _, _ := net.ParseCIDR(cidr)
	ipString := ip.String()
	splitted := strings.Split(ipString, ".")
	splitted[len(splitted)-1] = "1"
	gateway := strings.Join(splitted, ".")
	log.Println("Default gateway:", gateway)
	return gateway
}

func (wm *Wireman) TestTicker() {
	interval := 1 * time.Minute
	wm.Test()

	for range time.Tick(interval) {
		wm.Test()
	}
}

func (wm *Wireman) Test() {
	client := http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Get("http://icanhazip.com/")

	if err != nil {
		log.Println("Failed to test connection:", err)
		return
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Println("Failed to test connection:", err)
		return
	}

	publicAddress := strings.TrimSpace(string(body))
	log.Println("Wireguard connected on IP:", publicAddress)
}