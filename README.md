# docker-wireguard

Docker Wireguard implementation using [wireguard-go](https://github.com/WireGuard/wireguard-go), [wgctrl](https://github.com/WireGuard/wgctrl-go) & [netlink](https://github.com/vishvananda/netlink).

## Try it out

```
docker pull nxie/wireguard
```

## Example usage

```
services:
  test:
    depends_on:
      wireguard:
        condition: service_healthy
    image: alpine
    network_mode: container:wireguard
  wireguard:
    image: nxie/wireguard
    container_name: wireguard
    healthcheck:
      test: bash -c "[ -f /tmp/wireguard.lock ]"
      interval: 1s
      timeout: 3s
      retries: 10
    cap_add:
      - NET_ADMIN
    environment:
      - MULLVAD_ACCOUNT=123
      - MULLVAD_COUNTRIES=nl,Germany
    volumes:
      - ./config:/config
      - /dev/net/tun:/dev/net/tun
    ports:
      - 51820:51820/udp
```

A random `.conf` file from `./config` is used when the container starts. Optionally pass a Mullvad account ID to automatically download your config files.
