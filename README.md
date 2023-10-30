# docker-wireguard

## Example usage

This image allows you to route containers through your Wireguard VPN.

```
services:
  test:
    depends_on:
      - wireguard
    image: alpine
    network_mode: container:wireguard
  wireguard:
    image: nxie/wireguard
    container_name: wireguard
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
